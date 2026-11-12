#!/usr/bin/env python3
"""Headless end-to-end quest smoke test against a live server.

Walks the real wire protocol: create account -> create character -> enter
world -> talk to Elanore -> accept Tutorial -> Kael -> Ekinu -> reward.

Not part of the shipped client; run manually:
    SOM_DB_FILE=/tmp/x.db WS_PORT=27112 ./goserver &
    python3 _e2e_quest_smoke.py
"""
import asyncio
import os
import sys
import time

sys.path.insert(0, ".")

from som_protocol import QuestCatalog, SoMClient, item_name  # noqa: E402

URL = os.environ.get("SOM_E2E_URL", "ws://127.0.0.1:27112/ws")
# Fresh character every run so the quest state is deterministic even when the
# server keeps its database between runs.
_SUF = time.strftime("%H%M%S")
ACCT = "qt" + _SUF          # account names are capped at 16 chars
PASS = "secret1"
NICK = "Q" + _SUF


class Recorder:
    """Captures WorldState events (installed after login)."""

    def __init__(self, client):
        self.events = []
        self.dialogs = []
        self.choices = []
        self.quests = {}
        client.world.emit = self._on

    def _on(self, ev):
        self.events.append(ev)
        t = ev.get("type")
        if t == "dialog":
            self.dialogs.append((ev.get("author"), ev.get("text")))
        elif t == "dialog_choice":
            self.choices.append(list(ev.get("choices") or []))
        elif t == "quest":
            self.quests[ev["quest_id"]] = ev["state"]


async def wait_for(rec, pred, timeout=6.0):
    loop = asyncio.get_event_loop()
    end = loop.time() + timeout
    while loop.time() < end:
        for ev in list(rec.events):
            if pred(ev):
                return ev
        await asyncio.sleep(0.02)
    raise AssertionError(f"timeout; events={rec.events!r} choices={rec.choices!r}")


async def wait_choice(rec, needle, timeout=6.0):
    """Wait until a ContextChoice containing ``needle`` has arrived."""
    loop = asyncio.get_event_loop()
    end = loop.time() + timeout
    while loop.time() < end:
        for ch in rec.choices:
            if any(needle in str(x) for x in ch):
                return ch
        await asyncio.sleep(0.02)
    raise AssertionError(f"timeout waiting for choice {needle!r}; got {rec.choices!r}")


async def main():
    cat = QuestCatalog("quests.json")
    assert cat.loadable, "quests.json missing"

    c = SoMClient(URL, nick=NICK)
    await c.connect()
    try:
        # Account may already exist from a previous run; either way log in.
        if not await c.create_account(ACCT, PASS):
            print("create_account returned False (likely already exists)")
        assert await c.login(ACCT, PASS), "login failed"

        chars = await c.list_characters()
        if not chars:
            assert await c.create_character(NICK), "create_character failed"
        await c.connect_character(NICK)

        rec = Recorder(c)
        await asyncio.sleep(0.5)   # let enter-world pushes settle

        # --- Elanore: menu -> Tutorial offer -> accept -------------------
        await c.interact(1)
        menu = await wait_choice(rec, "Tutorial")
        print("Elanore menu:", menu)

        idx = [i for i, x in enumerate(menu) if "Tutorial" in str(x)][0]
        await c.trigger_choice(idx)
        offer_choices = await wait_choice(rec, "接受任务")
        print("offer:", rec.dialogs[-1][0], "|", rec.dialogs[-1][1][:50], "...")
        print("choices:", offer_choices)

        await c.trigger_choice(0)
        ev = await wait_for(rec, lambda e: e["type"] == "quest")
        qid = ev["quest_id"]
        assert ev["state"] == 1, ev
        print("accepted:", qid, cat.name(qid))

        # Starter items (Water Bottle, Cactus Sour Candy, Cotton Shirt, Shorts)
        inv = c.world.inventory
        starters = {item_name(k): v for k, v in inv.items()}
        print("starter items:", starters)
        assert len([1 for v in inv.values() if v > 0]) >= 4, inv

        # --- Kael: talk stage auto-advances 2 -> 3 -----------------------
        await c.interact(6)
        ev = await wait_for(rec, lambda e: e["type"] == "quest"
                            and e["quest_id"] == qid and e["state"] == 3)
        print("after Kael -> state", ev["state"])
        print("Kael says:", rec.dialogs[-1][1][:60].replace("\n", " "))

        # --- Ekinu: final talk -> auto stage -> done ---------------------
        await c.interact(7)
        ev = await wait_for(rec, lambda e: e["type"] == "quest"
                            and e["quest_id"] == qid and e["state"] == 255)
        print("after Ekinu -> state", ev["state"])
        print("Ekinu says:", rec.dialogs[-1][1][:60].replace("\n", " "))

        inv = c.world.inventory
        boots = inv.get(586504915, 0)
        goggles = inv.get(906139156, 0)
        print("rewards:", item_name(586504915), boots, "|",
              item_name(906139156), goggles)
        assert boots == 1 and goggles == 1, inv
        print("GP:", c.world.self_gp, "EXP:", c.world.self_exp)

        # --- catalog cross-check -----------------------------------------
        print("catalog stage 2 objective:",
              cat.objective(qid, 2, c.world))
        print("catalog state label:", cat.state_label(255))

        # --- A Drop of Relief: item requirement is consumed on turn-in -----
        # Nathan wants a Water Bottle, which the Tutorial starter kit gives.
        water = 3288439794
        assert c.world.inventory.get(water, 0) == 1, c.world.inventory

        rec.choices.clear()          # ignore the Tutorial dialogue history
        rec.dialogs.clear()
        await c.interact(10)         # Nathan offers exactly one quest
        await wait_choice(rec, "接受任务")
        await c.trigger_choice(0)
        ev = await wait_for(rec, lambda e: e["type"] == "quest"
                            and e["quest_id"] == 1931438535 and e["state"] == 1)
        print("accepted:", ev["quest_id"], cat.name(ev["quest_id"]))
        # The requirement is already met, so Nathan offers the turn-in at once.
        await wait_choice(rec, "完成任务")
        await c.trigger_choice(0)
        ev = await wait_for(rec, lambda e: e["type"] == "quest"
                            and e["quest_id"] == 1931438535 and e["state"] == 255)
        print("A Drop of Relief -> state", ev["state"])
        print("Water Bottles left:", c.world.inventory.get(water, 0))
        assert c.world.inventory.get(water, 0) == 0, c.world.inventory
        print("GP:", c.world.self_gp)

        print("\nE2E QUEST SMOKE OK")
    finally:
        await c.close()


if __name__ == "__main__":
    asyncio.run(main())
