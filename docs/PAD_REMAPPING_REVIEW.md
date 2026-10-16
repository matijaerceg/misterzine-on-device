# Unified controller implementation review

Invitations require a fresh physical press and are not queued during setup.
The short invitation uses one-button begin / two-button defer, with keyboard Esc
appended to the defer hint only when available. Defer waits for held releases and
then another fresh press, rather than suppressing the device for the app session.
Keyboard role supports arrows/Enter/Esc and a tap-to-switch, hold-to-select fallback.
Enter chooses on release to permit two-button chord arbitration. Opening/starting
holds cannot also select or assign. Separate keyboard Esc works during basics.
Keyboard role is asked first; typing acceptance skips setup. Manual entry always
requires six assignments. The active controller flow is six assignments, then a
physical-input table with action/name popups. Earlier full walkthrough, skip/pause
menus, proposed-map acceptance, standalone naming editor and preset UI are retired.
One-second holds cancel the six-step flow directly; the pause menu is retired.

MiSTercade V2 (`16d0:1358`) Generic HID firmware reports each direction as both
a button (Linux 704..707) and a hat edge (800..803). With the corresponding
capabilities present, setup gestures count each pair as one physical control and
wait for both releases. New basic assignments use the button code, matching the
vendor's global MiSTer map. The table retains both raw observations; holding a
direction edits its assigned member, including existing hat-based profiles.
Other models and real multi-control chords retain their previous behavior.
The regression reproduces the firmware's duplicate reports without hardware;
it does not establish a physical cabinet validation.

Profiles retain schema 1 for the file container; mapping_version 2 identifies the
new input-code-to-action dictionary. Many inputs may target one action. Legacy
fields remain decoding inputs only, converted by normalizedControl when hardware
is available. Conversion is not written until an explicit save succeeds. Names
are materialized to raw inputs, including old styles and action-bound names.
Existing Menu chords are retained by legacy_menu_chord until explicit Menu
assignment. Basic completion and keyboard/panel role are persisted per instance.
Raw source identity, kernel-held checks, capture ownership and echo suppression
remain in the platform reader. Versioned runtime mappings aggregate action holds.

The check table uses independent 200ms scroll interpolation with main-list bounded
centering and the host 60Hz animation path. No manual scrolling. Every held input highlights independently;
new presses redirect scrolling without restarting on repeats. One-second edit/exit hold arbitration
requires release across transitions. Popups use an independent navigation repeater.
All writes are atomic; failure keeps live settings unchanged and permits retry.

Button mapping graduated from beta to free on 2026-10-16 and its code moved into
this repository. Global label defaults and legacy OK preferences remain readable
and still apply to unmapped pads; the Options rows OK button and Button labels are
gone, replaced by Button mapping for everyone. Keyboard use is a
separate Options entry when keyboard-class devices are attached, with an explicit
chooser for multiple keyboards. No publication is included.

Regression coverage includes first setup, required controls, duplicate actions,
aggregate releases, typing roles, immediate names/defaults, failed save retry,
legacy conversion, independent physical profiles, stale/cancelled/entry-held input,
per-device deferral, centered scroll, multiple holds, and redo cancellation.
Harness captures (tools/render_controls.py) cover landscape/portrait at insets 15/40.
Hardware-specific encoder shift combinations still require physical testing.

Invitation holds invalidate the rendered frame during the host animation loop.
The regression test paints between ticks without forced screenshot invalidation,
checks advancing pixels for begin/defer and keyboard choices, and clears progress
on release. Default direction names follow the assigned direction (Up/Down/Left/
Right), including unknown axes and repurposed keys/buttons. Explicit physical
custom names still override defaults; resetting a name restores the direction.

Basic prompt order is Right, Left, Down, Up, OK, Back. The walkthrough has one
animated cancellation hint: Hold button on any pad > Cancel. Keyboard Esc remains
an escape without extra hint text. The table shows Hold any button > Edit it and
Hold any two buttons > Exit; edits still save immediately. Removing the third
hint restores table space. Popup label editing is offered only for actions whose
name can reach a rendered legend (controlLabelEditable): legend tokens including
Page up/down, never directions, Menu or unassigned inputs. Existing saved names
remain intact and label editing becomes available after reassignment.

The table reads Code | MiSTer/key | MZ label | Action. Noneditable MZ label cells
are muted even while the row is held; Menu shows N/A there, and a discovered
unassigned input shows -- for both MZ label and Action. Unassigned reads -- in
the action picker and the Latest line too.

Device numbers and the per-input Device settings submenu are removed. Redo uses
Options > Button mapping and cancellation returns to Options. Keyboard use saves
the role without changing assignments, or requires basics for an unproven panel.
Paired action edits (OK/Back, Left/Right, Up/Down) explicitly preview both physical
inputs before saving. Multiple tested targets require target selection; untested
targets are excluded. Names stay with their physical codes. Non-paired edits keep
the last-proven-input guard. Save failures keep the popup and previous profile.
