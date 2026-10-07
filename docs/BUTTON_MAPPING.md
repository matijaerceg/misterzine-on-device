# Controller setup and check (beta)

Enable **Show beta features**, then open **Options → Controls → Button mapping**.
Controller setup is a free beta: no supporter code is required. Public-source
builds retain inert hooks; the implementation is provided by the private overlay.
These mappings affect MisterZine only, never MiSTer's global settings or games.

## Six basic controls

An unfamiliar device is invited only when you press an input on it. That opening
press is consumed; release it before answering. Devices used during another setup
are not queued: press again after closing setup. Cancelling defers only that device
until its next fresh press, after all held inputs have been released. Repeats do
not reopen the invitation.

The short controller invitation shows its name and two instructions:
**Begin setup: hold one button** and **Not now: hold two buttons on any device**.
When a keyboard is connected, **or press Esc on keyboard** joins the Not now hint.
The opening press cannot begin setup; the hold that begins cannot assign Right.
A partial two-button release never becomes a single-button confirmation.

Keyboard-class devices first ask **Keyboard** or **Controller / pad**. Choose Keyboard
to retain ordinary navigation and typing without remapping. Use arrows, Enter and
Esc for this question, or tap a button to switch choices and hold one to choose.
Holding two buttons defers. These fallback controls also work on keyboard encoders. Choosing Controller / pad starts the six basic assignments;
unassigned keys then do not type. A separate keyboard can still type normally.

New controllers and every manual **Button mapping** entry assign **Right, Left,
Up, Down, OK, Back**. Press and release to advance. Duplicate and reserved inputs
are rejected. All six are required; progressing proves them. Recent completed
assignments appear as code:action pairs. Compatible optional mappings are retained
unless a basic assignment displaces one.

**Hold an input for one second to cancel directly.** Manual entry returns to
Options; an invitation returns to the interrupted screen. A hold on another
device also cancels, as shown by the on-screen hint. Esc on a separate keyboard
cancels immediately. Esc on the keyboard being mapped can still be assigned. Cancellation/disconnection
preserves the previous setup. The six assignments save together before Controller
check opens. Failed saves keep the previous setup and allow retry.

## Check, assign, and name inputs

The table shows **Code | Action | MiSTer/key | MZ label**. Code identifies the raw
input. MiSTer/key is its original reported slot or key; MZ label is the name in
MisterZine's button hints. Missing original mappings show a dash. Directions
default to Up, Down, Left, and Right, even for otherwise unnamed axes or repurposed
buttons. An explicitly chosen custom name still overrides the default.

Each physical input has its own row. Actions can have several inputs, or an empty
placeholder when unassigned. Required controls start confirmed. Other assigned
inputs are dim until pressed; newly discovered unassigned inputs appear below.
Pressing an input highlights it immediately and smoothly scrolls its row toward
the middle, stopping at the list ends. There is no manual scrolling. Every held input stays highlighted, including off-screen rows. A fresh press
redirects scrolling toward that input; repeats do not restart animation. Latest reports the most
recent input and its action, including unassigned inputs.

**Hold one input** to open its popup:

- **Change action:** assign it to an action or Unassigned. Existing buttons for
  that action stay assigned. You cannot remove the last tested input for a
  direction, OK, or Back; assign and test a replacement first.
- **Change OSD label:** pick a short name from categorized grids, including
  letters, numbers, PlayStation symbols, colors, fighting-game abbreviations,
  controller names and keyboard keys. **Use default name** resets only this input.
- **Device settings:** redo the six basic controls, or change a keyboard's role.

Popups use D-pad/OK/Back. Selecting an action or name saves immediately and
returns to the table. Back closes the popup without changing anything. Failed
saves leave it open for retry and retain the previously saved settings.

**Hold two inputs** on the table to exit. This does not save anything: successful
edits were already saved. Holds take one second. Holding a button on another device also exits.
Hold progress lights up the relevant instruction text.
Held inputs must be released before they can act on a newly opened screen.

The table consumes input without navigating the app, launching games or typing.
The screensaver stays out of setup/checking and gets a fresh idle period on exit.

## Existing setups and recovery

Existing valid profiles remain familiar. Old custom names, label styles and
OK/Back preferences retain their appearance and behavior when converted. Names
belong to physical inputs and survive action changes. The old separate naming,
preset, and OK-button screens are replaced by this flow in builds with controller
support; the global Button labels preference remains available.

Existing MiSTer Menu chords remain compatible. Assigning an explicit Menu input
replaces that inherited chord. The table identifies an inherited chord separately.
F12 is reserved. Devices without safely readable raw inputs must be configured
through MiSTer's controller setup; support is not universal.

Profiles live in `/media/fat/misterzine/controls.json`. Back up this file before
resetting or rolling back a build. New fields are versioned; incompatible or
damaged files are preserved rather than overwritten. Turning beta features off
retains saved profiles but disables their mappings and labels.

Physical identity uses a hardware identifier when available, otherwise the USB
connection path. Moving a port-backed device can produce a new invitation.
Receiver interfaces share an identity. Device 1, Device 2, etc. are temporary
display numbers. Compatible model profiles supply copied defaults, never shared
mutable preferences; unfamiliar physical devices still complete the six prompts.

Background: [J-PAC issue #5](https://github.com/matijaerceg/misterzine-on-device/issues/5).
Real keyboard encoders, shifted inputs and unusual controllers require hardware
checks; automated input simulations do not establish universal compatibility.
