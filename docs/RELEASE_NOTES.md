**Return after game now works from Console Mode**

MisterZine opened through Scripts -> MisterZine-Run, which is how Console Mode's Load Script and Degauss's Scripts list open it, now follows the games it starts, just as it does from the main-menu entry. The exit chord leaves the game, and with **Return after game** on, MisterZine reopens on the game you were playing. Thanks to RonanGH for reporting it.

For example: open MisterZine from Console Mode's Load Script, start a game, hold the exit chord, and you are back in MisterZine on that game.

**Fixed**

- Games started from a MisterZine opened through Scripts -> MisterZine-Run answer the exit chord and Return after game.
- Versions a pack keeps in a folder inside a game's alternatives folder, such as the trainers in Insert-Coin's Cave CV1000 pack, show up as versions of their game. Thanks to PhiLongTran for reporting it.
- Scrolling right after launch is smooth even while MisterZine is still reading the card.
- The user guide no longer says Reset or Exit in the OSD leaves a game; Reboot on the OSD's System page does.

Full details in the [changelog](https://github.com/matijaerceg/misterzine-on-device/blob/main/CHANGELOG.md#v124--2026-10-09).
Existing installations update through Update All, or through Update MisterZine only. New installation:
[installation guide](https://github.com/matijaerceg/misterzine-on-device#install-once).
