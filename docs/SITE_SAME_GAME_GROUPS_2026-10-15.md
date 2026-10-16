# Site handoff: same-game umbrella rows (2026-10-15)

Site commits b17d946 + 3389fa9 (misterzine repo: misterzine.py and
docs/releases/index.html). Live since the 2026-10-15 evening refresh.
Data contract: data.json rows gained ONE optional field, `g`. Nothing else
changed. The app ignores unknown JSON fields, so it keeps working untouched;
this doc is for the session that decides whether the app mirrors the site.

## The new field

- `g` (string) appears only on arcade rows that belong to a same-game group.
  Its value is the group's shared one-hop MAME parent setname (e.g.
  `bublbobl`, `galivan`, `tmnt2`). Rows without `g` are not grouped.
- Same game = same MAME setname OR same one-hop parent (`meta[sn].parent or
  sn`, NOT the full clone chain) AND same base title (`_arcade_base` of `mt`
  or `title`, casefolded). The grouping is done in the pipeline
  (`attach_same_game_groups`), never client-side: title-only or family-only
  grouping gives false friends (Victory, Burger Time DECO, Tetris on three
  boards, Sprint 1/2). Do not re-derive groups in Go; read `g`.
- A group must span 2+ distinct (src, core) pairs. Same-core variant pairs
  (Asteroids Deluxe green/orange, Kickle Cubicle / Meikyu Jima) are NOT
  grouped. Mortal Kombat T-Unit / Y-Unit on blahm1d IS a group.
- At ship: 37 groups covering 83 rows, all pairs or triples. `g` is
  deterministic run to run.
- Member rows keep their own `k`, `title` ("Title (CoreLabel)"), dates,
  favorites and deep links. The umbrella itself has no `k`; the site's
  permalink for it is `#g:<g>` (prefix, because a member `k` can equal the
  root setname: bublbobl).

## What the site does with it (mirror only what makes sense on a TV)

- After filtering, members sharing `g` fold into one umbrella row: chevron,
  base title (shortest title of the earliest-debut member, parenthetical
  stripped), a "2 cores" / "3 cores" chip, Core cell = member core labels
  joined with " · " (a core label that equals the game title is replaced by
  its source name), Source cell = distinct source names.
- Sort values of the umbrella: max `updated` / `mz` / `b`, min `date`;
  DEBUT pill only if every member is debut-dated. Shared specs (genre, year,
  manufacturer, reg, rot, res, plr, ctl, spc, flip, sn, scr) are shown only
  when all members agree, else blank. Screenshot = first member with `img`.
- Filters hide members; a group left with ONE visible member is promoted to
  a plain row (no umbrella). Favorites-only shows starred members as plain rows.
- No favorite star on the umbrella. Favorites, since-visit dots, DEBUT, beta
  and deprecated badges stay on members.
- Click rule: closed umbrella -> open + select; open but a different row
  selected -> select the umbrella only (stays open); open and already
  selected -> collapse. Keyboard: Right expands, Left collapses, Up/Down walk
  umbrella then members (an expanded group is 1 + N stops).
- Selecting a member by any route (deep link, arrows, panel card) auto-expands
  its group and highlights the member.
- Group panel = shared facts once, then one card per member (own title,
  core, source, dates, star, launch row, "open" link to the member's panel).
- Tally still counts releases (members), not umbrellas.
- The in-table render uses a `flat` list (umbrella + members while expanded);
  every row index on the site points into `flat`, not the sorted top level.

## If the app adopts it

- `internal/data/row.go`: add `G string \`json:"g,omitempty"\``. Everything
  else in `data` (Ingest, sort, since-visit, filters) can stay per-member;
  grouping is a view-layer fold after filtering, exactly like the site.
- Local-card rows (ROM sweep / standin) carry no `g`; they stay plain rows.
  A catalogue member and a local row for the same setname is the existing
  merge rule's business, not the group's.
- The app's list is navigated with a d-pad, so the site's click rule maps
  to: A on a closed umbrella = open + move cursor onto it; A on the open,
  highlighted umbrella = collapse; Left/Right = collapse/expand. Decide
  whether expansion is remembered across sort changes (site: `expanded` set
  keyed by `g`, not persisted, survives live data swaps).
- Keep favorites and the launch target on members. An umbrella has nothing
  to launch.
- Details screen for an umbrella: shared block once, then a per-core stack,
  as the site's panel does.
- Goldens: the list and Details renders change, so follow
  docs/RELEASING.md and the render-goldens rule before pushing.

## Reference on the site side

- Pipeline: misterzine.py `attach_same_game_groups` (post-pass after
  `attach_arcade_patches` in cmd_export_web). Tests: SameGameGroupTests in
  test_card_status.py.
- Site: docs/releases/index.html `makeUmbrella`, `toggleGroup`, `groupHTML`,
  `G2ROW`, `ALLG`, `expanded`, `flat`; CSS `tr.grp`, `tr.member`, `.gc`.
- Live examples: https://misterzine.fyi/releases/#g:bublbobl (umbrella),
  https://misterzine.fyi/releases/#galivan (member deep link).
