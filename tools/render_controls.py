"""Button mapping renders: six-input setup, physical-input table and immediate editing.

Checked against testdata/render_golden_controls.json."""
import os
from pathlib import Path
import subprocess
os.chdir(Path(__file__).resolve().parents[1])
go=os.environ.get('GO','go')
basic='; '.join('raw 0 '+str(c) for c in [801,800,803,802,304,305])+'; '
ready='enter; '+basic
hold='rawpress 0 330; wait 1100; rawrelease 0 330; '
scroll='; '.join('raw 0 '+str(c) for c in range(330,352))+'; '
scripts={
 'no-label-edit':ready+'rawpress 0 802; wait 1100; rawrelease 0 802; shot direction; back; '+hold+'shot unassigned',
 'swap':ready+'rawpress 0 304; wait 1100; rawrelease 0 304; enter; down; enter; shot confirm; enter; shot swapped; rawpress 0 802; wait 1100; rawrelease 0 802; enter; down; enter; shot direction-confirm; back; shot cancelled',
 'swap-alias':ready+hold+'enter; home; down*5; enter; rawpress 0 304; wait 1100; rawrelease 0 304; enter; down; enter; shot targets; down; enter; shot chosen; enter; shot swapped',
 'direction-names':'enter; '+('; '.join('raw 0 '+str(c) for c in [334,335,336,337,338,339]))+'; wait 250; shot defaults',
 'basic':'enter; shot right; raw 0 801; raw 0 801; shot duplicate; raw 0 800; shot down; raw 0 803; shot up; raw 0 802; raw 0 304; shot history; rawpress 0 305; wait 500; shot holding; wait 600; rawrelease 0 305; shot cancelled',
 'table':ready+'wait 250; shot virgin; rawpress 0 315; wait 100; shot first-down; rawrelease 0 315; '+scroll+'wait 80; shot scrolling; wait 250; shot bottom; rawpress 0 802; wait 100; shot returning; rawpress 0 803; shot chord; wait 500; shot exit-holding; wait 600; rawrelease 0 802; rawrelease 0 803; shot exited',
 'actions':ready+hold+'shot button; enter; shot actions; end; up*2; shot launch; enter; wait 250; shot assigned; rawpress 0 304; wait 1100; rawrelease 0 304; enter; end; enter; shot required-protected; back; shot table',
 'labels':ready+'rawpress 0 304; wait 1100; rawrelease 0 304; '+'down; enter; shot default; down; right*2; shot playstation; down; right; enter; shot named; '+'rawpress 0 304; wait 1100; rawrelease 0 304; down; enter; enter; shot reset',
 'cells':ready+hold+'enter; end; up; enter; wait 250; shot menu; rawpress 0 802; rawpress 0 330; wait 100; shot held-muted; rawrelease 0 802; rawrelease 0 330; raw 0 331; wait 250; shot unassigned; rawpress 0 331; wait 1100; rawrelease 0 331; enter; shot picker',
 'keyboard':'down; enter; shot right; '+('; '.join('raw 1 '+str(c) for c in [106,105,108,103,28,1]))+'; shot table; unplug; wait 1100; shot disconnected',
 'defer':'enter; rawpress 1 28; wait 500; shot holding; wait 1100; rawrelease 1 28; shot deferred',
}
for rotation in ([],['-rot','left','-logical']):
 for inset in (15,40):
  for name,script in scripts.items():
   subprocess.run([go,'run','./cmd/mzharness','-controls','-show-beta',*rotation,'-inset',str(inset),'-out',f'out-controls/{name}-{"t" if rotation else "h"}-{inset}','-script',script],check=True)
for args,name in [([], 'hidden'), (['-show-beta'],'free-beta')]:
 subprocess.run([go,'run','./cmd/mzharness','-controls-auto',*args,'-out',f'out-controls/{name}','-script','back; home; pagedown*3; right; shot options'],check=True)
for rotation in ([],['-rot','left','-logical']):
 for inset in (15,40):
  for name,script in {
   'keyboard-use':'back; home; pagedown*3; right; shot options; down*2; enter; shot keyboard; down; shot panel; back; shot cancelled',
   'automatic':'wait 2200; shot waiting; raw 1 28; shot keyboard-choice; raw 1 30; shot controller-choice; rawpress 1 30; wait 500; shot role-hold; wait 600; rawrelease 1 30; shot keyboard-basic; rawpress 1 30; wait 1100; rawrelease 1 30; raw 0 304; shot invitation; rawpress 0 304; wait 500; shot begin-hold; wait 600; rawrelease 0 304; shot basic; raw 1 1; shot escaped',
   'invite-defer':'raw 0 304; rawpress 0 304; rawpress 0 305; wait 500; shot chord-hold; rawrelease 0 305; wait 1100; shot partial-chord; rawrelease 0 304; rawpress 1 30; rawpress 1 31; wait 1100; rawrelease 1 30; rawrelease 1 31; shot deferred; raw 0 304; shot reinvited; raw 1 1; shot escaped'
  }.items():
   subprocess.run([go,'run','./cmd/mzharness','-controls-auto','-show-beta',*rotation,'-inset',str(inset),'-out',f'out-controls/{name}-{"t" if rotation else "h"}-{inset}','-script',script],check=True)
