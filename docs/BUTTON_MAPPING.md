# Button mapping (beta)

Make MisterZine's controls and button hints match the pad or arcade cabinet you
are using. This feature is experimental, without the Supporter star. In a build
that includes it, enable **Options → Operation → Show beta features** and use
an October 2026 or later code. Open **Options → Controls → Button mapping**.

## Set up a controller

1. MisterZine opens the controller you are using, or the only directly readable
   pad. **Change controller** lets you pick another. Controllers of the same
   model share a saved setup.
2. **Set up buttons** walks through Launch, Open, Back, Filters and View.
   Each step lets you change the button or keep the current one. There is no
   countdown. Directions and extra buttons are under **Your buttons → More buttons**.
3. To change a button, press and release it. MisterZine shows your choice and
   explains any swap before you accept it. Choose **Try another button** to
   correct a mistake, or **Keep previous button** to undo that step. Holding
   a button for two seconds cancels capture; Esc on a separate keyboard also
   returns. Completed steps stay available to review.
4. **Test buttons** shows what each press would do without launching a game.
   It stays open until you hold the proposed Back button for two seconds or
   press Esc on a separate keyboard.
5. **Your buttons** is a compact summary. Choose any action to change it, or
   **Save changes** to finish. If the controls changed, press and release the
   new Open button, then the new Back button. Take as long as you need;
   holding a button for two seconds cancels this check. Label-only changes
   save directly.
6. **Buttons saved** confirms completion. Choose **Back to games** to play.

Your saved controls keep navigating setup until you finish saving. A disconnect,
interrupted input or failed write preserves them. Leaving with unsaved changes
lets you keep editing or discard. **More buttons → Restore defaults** asks before
restoring the original buttons and labels; review and save to apply the reset.

Launch, Open, Back, and the four directions must be assigned and distinct when
saving a custom mapping. Optional actions can be left unassigned. A pad arriving
through MiSTer's translated input must first be defined in MiSTer's menu; the
page explains this rather than offering a setup it cannot read reliably.

## Labels that match your panel

Open **Button labels**. **Label style** changes only with left/right, with arrows
showing the available directions; OK leaves it unchanged. A preview shows the
resulting Launch, Open and Back hints. Choose the existing global
labels, MiSTer letters, Xbox letters, PlayStation symbols, numbers, **Neo Geo
A B C D**, or **Custom letters**. Presets follow the physical MiSTer slot after
a remap, so moving Filters onto Start also changes the hint to Start.

Custom letters provide a separate label for each of the nine button actions,
including Launch. Left/right chooses **A–Z or 0–9**. For example, set Launch to
`Y` for a yellow start button, Open to `R` for red, and Back to `B` for blue.
Single-character labels keep the legends readable on a small CRT. They name
MisterZine actions, not a game's controls. Directions retain arrow hints.

## J-PAC and other keyboard encoders

Choose the encoder from the device list and change **Use as** to **Arcade panel**.
This is an explicit choice; a USB identity alone never converts a typing keyboard.
The starting layout is the common MAME keyboard layout:

| Control | Initial key |
|---|---|
| Launch | 1 |
| Open / confirm | Left Ctrl |
| Back / Options | Left Alt |
| Filters | Space |
| View / favorite | Left Shift |
| Page up / down | Z / X |
| Quick toggles | 5 |
| Menu | Esc |
| Directions | Arrow keys |

Use the guided setup if the encoder was reprogrammed. Save changes to the panel
choice before using it on the game list. Once saved, keys from that model act
as controls and stop entering search text; a separate keyboard model can still
type. Change Use as back to **Typing keyboard** and save to undo the mode.
Restore defaults keeps the chosen keyboard/panel mode.

Panel recognition from MiSTer.ini and importing MiSTer's keyboard-encoder maps
are not part of this first beta. This manual route addresses the Start/search
problem described in [issue #5](https://github.com/matijaerceg/misterzine-on-device/issues/5).
The earlier [controls guide](USER_GUIDE.md#controls) explains the underlying
MiSTer slots, OK-button choice, and existing global label styles.

## Recovery and scope

All changes belong to MisterZine. Game mappings, MiSTer's global definitions,
and the in-game exit chord are separate. A MiSTer-defined two-button Menu combo
remains available. F12 keeps its existing host function and cannot be assigned.

Turning **Show beta features** off stops these mappings and labels and preserves
them for later. The separate `/media/fat/misterzine/controls.json` holds the
profiles; public-source builds leave it alone. To recover outside the app, close
MisterZine and rename that file, then restart. A damaged or newer-format file is
left intact and the screen explains that it cannot save over it.

Keyboard encoders remain shared with MiSTer so their existing menu combinations
can work. MiSTer's source-less translated controller events are ignored while a
configured panel is connected, as they already are while a directly read pad is
connected. Use directly readable pads alongside a panel. Actual encoder shift
combinations and game return behaviour still need testing on hardware.

## Hardware review

- On a pad, swap Launch with Filters. Check the swap explanation and try again
  or keep the previous button. Test the choices, cancel the save check with a
  held button, and confirm the old mapping still works. Repeat and save.
- Leave setup idle for a minute: it must not skip a step or discard work. Verify
  Label style changes only with left/right and shows arrows and preview hints.
- Restart MisterZine and check the mapping and legends. Try a second controller.
- Choose Neo Geo and custom labels; inspect the list, Details and Options in
  landscape and tate, including a large safe zone.
- Disconnect during capture and during each save-test step. Reconnect and
  verify that no unfinished mapping was saved.
- On a real J-PAC/I-PAC, test factory and reprogrammed keys, held modifiers,
  shifted menu combinations, a separate typing keyboard, and game launch/return.
- Turn beta features off and on, then restore defaults. Confirm the saved
  preferences survive a temporary return to a build without this feature.
