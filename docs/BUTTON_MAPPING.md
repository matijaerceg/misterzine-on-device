# Button mapping (beta)

Make MisterZine's controls and button hints match the pad or arcade cabinet you
are using. This feature is experimental, without the Supporter star. In a build
that includes it, enable **Options → Operation → Show beta features** and use
an October 2026 or later code. Open **Options → Controls → Button mapping**.

## Set up a controller

1. Choose the controller from the connected-device list. The USB identity is
   shown beneath its name. Controllers of the same model share a saved setup.
2. Choose an action, then press and release the physical button you want to use.
   **Set up all buttons** walks through the actions and directions in order.
   Waiting eight seconds keeps the current assignment. Esc on a separate
   keyboard cancels capture.
3. A button already assigned to another action swaps places with that action's
   old button. An asterisk marks edited assignments. Your saved controls keep
   navigating the editor while you work on the draft.
4. **Test draft buttons** shows what each press would do, without launching a
   game or navigating away. Hold the draft's Back button for two seconds, use
   Esc on a separate keyboard, or wait 30 seconds without a press to return.
5. **Save & test** asks you to press and release the new **Open / confirm** button,
   then the new **Back / Options** button. Each step has a 12-second limit.
   Both must be demonstrated on the selected device before anything is saved.

Waiting, disconnecting the controller, interrupted input, or a failed write
keeps the current saved setup. A timeout leaves the draft available to correct.
Leaving with unsaved edits offers **Discard** or **Keep editing**. **Restore
defaults** resets the draft's buttons and labels; use Save & test to apply it.

Launch, Open, Back, and the four directions must be assigned and distinct when
saving a custom mapping. Optional actions can be left unassigned. A pad arriving
through MiSTer's translated input must first be defined in MiSTer's menu; the
page explains this rather than offering a setup it cannot read reliably.

## Labels that match your panel

**Label style** applies to the selected controller. Choose the existing global
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

Use the guided setup if the encoder was reprogrammed. Save & test the panel
choice before using it on the game list. Once saved, keys from that model act
as controls and stop entering search text; a separate keyboard model can still
type. Change Use as back to **Typing keyboard** and Save & test to undo the mode.
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

- On a pad, swap Launch with Filters. Test the draft, let the save trial expire,
  and confirm the old mapping still works. Repeat and save with the new buttons.
- Restart MisterZine and check the mapping and legends. Try a second controller.
- Choose Neo Geo and custom labels; inspect the list, Details and Options in
  landscape and tate, including a large safe zone.
- Disconnect during capture and during each save-test step. Reconnect and
  verify that no unfinished mapping was saved.
- On a real J-PAC/I-PAC, test factory and reprogrammed keys, held modifiers,
  shifted menu combinations, a separate typing keyboard, and game launch/return.
- Turn beta features off and on, then restore defaults. Confirm the saved
  preferences survive a temporary return to a build without this feature.
