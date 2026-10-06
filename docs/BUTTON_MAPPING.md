# Button mapping (beta)

Check what your controller already does, then accept it or give every function
its own button. These controls belong to MisterZine; game controls and MiSTer's
global controller definitions are separate.

This ordinary beta feature has no Supporter star. In a build that includes it,
enable **Options → Operation → Show beta features** and unlock the beta with an
October 2026 or later code.

## Check a controller

An unfamiliar controller or keyboard opens **New controller detected** when the
app is ready. Updates, launching and other screens are not interrupted. Accepted
models and models with existing saved profiles do not prompt again. Identical
models share their preferences.

For another check, open **Options → Controls → Button mapping** with the controller
you want to configure. The screen says **Currently mapping:** and its name. When
there is no identifiable opening controller, choose one from the list.

The functions start untested. Press buttons to populate their rows; held inputs
light up. Unmatched inputs appear separately, and the bottom line names the most
recent input. Presses here do not navigate, search or launch games.

- **Hold one button for two seconds, then release:** accept the existing setup.
- **Hold two buttons together for two seconds, then release both:** remap all.

Both gestures remain available after any number of test presses.
Directions do not count toward these gestures. The progress bar only appears
while holding; there is no timeout. Adding a second button cancels single-button
acceptance. Buttons already held on entry must first be released. Interrupted
input and disconnects cannot confirm anything.

To leave an inaccessible controller, **hold any button for two seconds on another
controller or keyboard, then release**. This discards any unfinished setup without accepting
the selected controller. Further automatic invitations wait until the next app
start; manual Button mapping remains available.

## Remap everything

Setup immediately asks for **Up**, then **Down, Left, Right, OK, Back, Launch,
Filters, View/favorite, Page up, Page down, Quick toggles and Menu**. Each step
explains the function and suggests a typical physical button. Completed choices
remain visible. Tap and release to assign and advance.

Each input can be assigned once. If it is already used, the screen names that
function and waits for another choice; nothing swaps automatically.

**Hold any button for two seconds** to pause. Tap a button to cycle through
**Resume · Skip · Cancel setup**, then hold for two seconds and release to select.
These controls work before navigation is mapped. The four directions, OK and
Back cannot be skipped. Other functions, including Launch, may be skipped;
**skipped means unassigned**, not inherited from the old setup.

Cancel returns directly to the diagnostic with the saved setup. Completing the
final function saves the mapping immediately and returns to the ordinary check
screen. **Controls saved** briefly confirms success. Hold one button and release
to accept and close; hold two and release both to start remapping again. These
gestures are the same on every visit. Check rows populate on button-down, and
all currently held assigned inputs highlight together.

Nothing changes before all functions are assigned or explicitly skipped. The
four directions, OK and Back remain required by save validation. A failed save
keeps the previous setup and the completed choices, with a hold-to-retry prompt.
The screensaver stays out of setup and gets a fresh idle period on exit.

## Keyboards and arcade panels

Accepting a keyboard as it is preserves ordinary keyboard navigation and typing:
Enter, Esc, Tab, Space, arrows, page keys and existing search behaviour.
Untested keyboard rows show the expected key in parentheses; pressing it confirms
what actually arrives. A missing release cancels hold progress when the physical
key is no longer held, so a later tap cannot accept a stale hold.

Completing and saving a full keyboard remap explicitly makes that model an arcade
panel. Its unassigned keys no longer enter search. A separate keyboard model can
still type. Existing saved panel profiles retain their layout and labels.

J-PAC and similar encoders can use this flow when their physical key inputs are
readable. USB identity alone does not convert a typing keyboard into a panel.
Encoder-specific shift combinations and unusual hardware need individual testing.
For devices that cannot be captured safely, use **MiSTer's controller setup** first;
the diagnostic explains the limitation. This is not a promise of universal support.

## Labels

The separate **Options → Controls → Controller labels** screen retains global,
MiSTer, Xbox, PlayStation, numeric, **Neo Geo A B C D**, and custom letter styles.
Left/right changes the style or custom character; arrows indicate available
changes. OK does not cycle styles. Save changes to apply labels.

Custom letters name actions, for example `Y` for a yellow Launch button or `R`
for a red OK button. Labels remain separate from the remapping walkthrough.

## Recovery and background

Profiles are stored in `/media/fat/misterzine/controls.json`. Turning beta features
off disables their mappings and labels while retaining the file. Public-source
builds leave these features inert. To reset outside the app, close MisterZine,
back up and rename that file, then restart. Damaged or newer-format files are
preserved and cannot be overwritten through setup.

Existing setups retain MiSTer's two-button Menu chord. A full remap replaces
Menu too; skipping it leaves no Menu action. F12 cannot be assigned. Keyboard encoders remain shared with MiSTer; source-less translated
echoes are suppressed during diagnostics and while a configured panel is connected.

Background: [J-PAC request, issue #5](https://github.com/matijaerceg/misterzine-on-device/issues/5)
and the [existing controls guide](USER_GUIDE.md#controls). Neo Geo and custom
letter labels address cabinet button legends independently of input mapping.

## Hardware review

Check first detection, accept/reconnect, manual entry from each controller, and
ordinary keyboard typing. Try late second-button presses, staggered releases,
buttons held on entry, disconnects and a long idle pause. Remap every function,
try a duplicate, skip optional actions, cancel, retry, and save. Confirm that
unmatched inputs never launch, navigate or type in the diagnostic. Inspect labels
and both landscape and portrait layouts. Real keyboard encoders additionally need
shifted keys, a second typing keyboard, and game launch/return checked on hardware.
