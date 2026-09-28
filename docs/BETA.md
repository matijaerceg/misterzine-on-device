# MisterZine Arcade (Patreon beta)

MisterZine Arcade is the members' build of MisterZine for supporters at
[patreon.com/MisterZine](https://www.patreon.com/MisterZine). It is the free
MisterZine plus the features still being finished, and it takes the free
version's place on your card: favorites and settings come along, and Update
All keeps it current. You can go back to the free version at any time.

## Install

Your MiSTer needs its network connection and Downloader, which Update All
installs.

1. Download **MisterZine-Install-Beta.sh**, attached to the newest beta on the
   [releases page](https://github.com/matijaerceg/misterzine-on-device/releases)
   and linked from the members' post.
2. Copy it to the `Scripts` folder on your SD card.
3. On the MiSTer, open Scripts and run **MisterZine-Install-Beta**.
4. Return to the main menu, choose **MisterZine Arcade** and enter the code
   from the members' post.

The installer works whether or not MisterZine is already on the card, and
running it again is safe. It changes one line of your Downloader settings: the
address of the `misterzine` entry, which now leads to the beta. Your other
Downloader settings stay as they are. On a card that never had MisterZine it
also adds the main-menu entry, as **MisterZine-Setup** does.

## The code

MisterZine Arcade asks for the six-digit code from the current members' post
and remembers it. A new batch of features comes with a new code: after that
update the app asks again, and the newest post has it. Please keep the code
to yourself.

## Updates

New betas arrive with your normal Update All runs. When one is out, the bar
at the bottom of the list shows **App update**, and **Update MisterZine only**
in Options fetches just the new beta in a few seconds.

## Back to the free version

Run **MisterZine-Switch-To-Free** from Scripts. It points the `misterzine`
entry back at the free releases, installs the free MisterZine in the beta's
place and removes the MisterZine Arcade menu entry; **MisterZine** returns.
Favorites and settings stay. Settings that only the beta has are left behind.

The code screen offers the same switch: press X twice there. The app shows
the switch as it runs, and when it has finished, A restarts into the free
MisterZine.

Keep **MisterZine-Install-Beta** in Scripts to come back later.

## If something goes wrong

- *Downloader is not on this card*: run Update All once, then run the script
  again.
- *An updater is running*: let Update All or Downloader finish, then try again.
- *Downloader did not finish*: check the network connection and run the
  script again. Update All also finishes the switch, since the `misterzine`
  entry already names the version you chose.

**MisterZine-Uninstall** removes MisterZine Arcade the same way it removes the
free version. For anything else, see the
[user guide](USER_GUIDE.md) and [troubleshooting](TROUBLESHOOTING.md).
