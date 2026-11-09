package main

// Phase-3 combat loopback test: TriggerSkill -> Casted + TargetAlteration ->
// death -> TriggerRespawn -> WarpPlayer + FullUpdateEntity.
//
// The test uses two fake clients in the same AOI. A casts a high-damage magic
// skill (Flar) on B from spawn distance (no movement needed), verifying the
// full damage/death/respawn flow in just a few casts. A dedicated goroutine
// drains each client's pushes into a buffered channel so the test never fights
// over websocket read deadlines (the cause of earlier flaky failures).

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// flarSkillID is the SoM "Flar" magic skill (presets/cells/skills/Flar.tres):
// Attack=50, Range=120, Magic=true. Reach = 32+120+32 = 184 > spawn distance.
const flarSkillID = 2089110122

// pushCollector drains all server pushes from a fake client into a channel.
// It stops when the client disconnects or the channel is full (256).
type pushCollector struct {
	ch     chan *ParsedRPC
	fc     *fakeClient
	done   chan struct{}
	closed bool
}

func startCollector(fc *fakeClient) *pushCollector {
	pc := &pushCollector{
		ch:   make(chan *ParsedRPC, 256),
		fc:   fc,
		done: make(chan struct{}),
	}
	go func() {
		for {
			// Use a generous deadline; the collector runs for the whole test.
			fc.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			_, data, err := fc.conn.ReadMessage()
			if err != nil {
				close(pc.ch)
				return
			}
			cmd, rpc, _, _, err := ParsePacket(data)
			if err != nil || cmd != CmdRemoteCall || rpc == nil {
				continue
			}
			select {
			case pc.ch <- rpc:
			default:
				// Channel full; drop oldest to avoid blocking the reader.
			}
		}
	}()
	return pc
}

// recv waits for a push with the given nameID within the timeout.
func (pc *pushCollector) recv(nameID uint16, timeout time.Duration) *ParsedRPC {
	deadline := time.After(timeout)
	for {
		select {
		case rpc, ok := <-pc.ch:
			if !ok {
				return nil
			}
			if rpc.NameID == nameID {
				return rpc
			}
		case <-deadline:
			return nil
		}
	}
}

// TestCombatLoopback verifies the full combat flow between two simulated
// clients: cast broadcast, damage application, death, and respawn.
func TestCombatLoopback(t *testing.T) {
	url, stop := startTestServer(t)
	defer stop()

	a := dialFake(t, url)
	a.announce(t, 5, "/root/Network")
	b := dialFake(t, url)
	b.announce(t, 5, "/root/Network")

	enterPartyWorld(t, a, "cuser1", "Cattacker")
	enterPartyWorld(t, b, "cuser2", "Ctarget")

	// Start collectors after enter-world so the enter-world pushes are consumed
	// by enterPartyWorld's own read loop.
	colA := startCollector(a)
	colB := startCollector(b)

	// Flar reach = attackRange(32) + skillRange(120) + 2*radius(16) = 184.
	// A and B spawn 84.85 px apart (same spawn grid), so no movement is needed.
	// A casts Flar on B repeatedly until B dies. Each cast produces:
	//   A: Casted, TargetAlteration
	//   B: TargetAlteration
	// On the killing blow, B also gets FullUpdateEntity(state=DEATH), and A
	// gets the same death broadcast.
	death := false
	for i := 0; i < 10 && !death; i++ {
		pkt, _ := BuildRPCPacket(5, rpcTriggerSkill, []Variant{NewInt(2), NewInt(flarSkillID)})
		a.conn.WriteMessage(websocket.BinaryMessage, pkt)

		// A gets Casted (inclusive) and TargetAlteration (inclusive).
		casted := colA.recv(rpcCasted, 2*time.Second)
		if casted == nil {
			t.Fatalf("cast %d: A did not receive Casted", i)
		}
		if casted.Args[0].Int != 1 || casted.Args[1].Int != flarSkillID {
			t.Fatalf("Casted: want attacker=1 skill=%d, got %s", flarSkillID, formatArgs(casted.Args))
		}
		alt := colA.recv(rpcTargetAlteration, 2*time.Second)
		if alt == nil {
			t.Fatalf("cast %d: A did not receive TargetAlteration", i)
		}
		if alt.Args[0].Int != 1 || alt.Args[1].Int != 2 {
			t.Fatalf("TargetAlteration: want attacker=1 target=2, got %s", formatArgs(alt.Args))
		}
		// alteration must be HIT(0), CRIT(1), or DODGE(3)
		if alt.Args[3].Int != 0 && alt.Args[3].Int != 1 && alt.Args[3].Int != 3 {
			t.Fatalf("TargetAlteration alteration: want 0/1/3, got %s", formatArgs(alt.Args))
		}
		if alt.Args[3].Int != 3 && alt.Args[2].Int <= 0 {
			t.Fatalf("TargetAlteration: non-dodge must deal >0 damage, got %s", formatArgs(alt.Args))
		}

		// B also gets TargetAlteration (B is in A's AOI).
		altB := colB.recv(rpcTargetAlteration, 2*time.Second)
		if altB == nil {
			t.Fatalf("cast %d: B did not receive TargetAlteration", i)
		}
		if altB.Args[0].Int != 1 || altB.Args[1].Int != 2 {
			t.Fatalf("B TargetAlteration: want attacker=1 target=2, got %s", formatArgs(altB.Args))
		}

		// On the killing blow, B receives a FullUpdateEntity with state=DEATH.
		// Drain any remaining pushes briefly and check for the death broadcast.
		time.Sleep(50 * time.Millisecond)
		for {
			pr := colB.recv(rpcFullUpdateEntity, 100*time.Millisecond)
			if pr == nil {
				break
			}
			if pr.Args[4].Int == stateDeath {
				death = true
			}
		}
	}
	if !death {
		t.Fatal("B did not die after 10 Flar casts")
	}

	// A gets the death broadcast (inclusive AOI).
	deathPush := colA.recv(rpcFullUpdateEntity, 2*time.Second)
	if deathPush == nil {
		t.Fatal("A did not receive death broadcast")
	}
	if deathPush.Args[4].Int != stateDeath {
		t.Fatalf("A death broadcast: want state=DEATH, got %s", formatArgs(deathPush.Args))
	}

	// B is dead: TriggerRespawn -> WarpPlayer + FullUpdateEntity (revived).
	pkt, err := BuildRPCPacket(5, rpcTriggerRespawn, []Variant{})
	if err != nil {
		t.Fatalf("build respawn: %v", err)
	}
	b.conn.WriteMessage(websocket.BinaryMessage, pkt)
	warp := colB.recv(rpcWarpPlayer, 2*time.Second)
	if warp == nil {
		t.Fatal("B did not receive WarpPlayer")
	}
	if warp.Args[0].Int != 0 {
		t.Fatalf("respawn WarpPlayer: want map=0, got %s", formatArgs(warp.Args))
	}
	revive := colB.recv(rpcFullUpdateEntity, 2*time.Second)
	if revive == nil {
		t.Fatal("B did not receive respawn FullUpdateEntity")
	}
	if revive.Args[4].Int != stateIdle {
		t.Fatalf("respawn FullUpdateEntity: want state=IDLE(0), got %s", formatArgs(revive.Args))
	}
	// A sees the respawn too (AOI broadcast).
	reviveA := colA.recv(rpcFullUpdateEntity, 2*time.Second)
	if reviveA == nil {
		t.Fatal("A did not receive respawn broadcast")
	}
	if reviveA.Args[4].Int != stateIdle {
		t.Fatalf("A respawn broadcast: want state=IDLE(0), got %s", formatArgs(reviveA.Args))
	}
}
