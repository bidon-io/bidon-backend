# Stacked PRs with git-spice

We stack dependent branches instead of landing one giant PR or waiting on a
chain of manual rebases. [git-spice](https://abhinav.github.io/git-spice/)
(`gs`) tracks the stack and handles restacking, submission, and merged-branch
cleanup.

Trunk is **`new-main`**, not GitHub's default branch (`main`). All active
`bac-*` work branches off `new-main`, and `gs repo init` is configured
accordingly.

Branch names follow `bac-NN-slug`, matching the Linear issue's `gitBranchName`
and git-spice's own auto-generated names (derived from a `BAC-NN: Title`
commit subject), so no `spice.branchCreate.prefix` is needed.

## Tooling

`git-spice` and `gh` come from the Nix devShell (`flake.nix`). Inside the
devShell, `gs` and `gh` resolve to these versions and shadow whatever's on
your system `$PATH` — including Ubuntu's Ghostscript, which also installs a
`gs` binary. Outside the devShell, `gs` is Ghostscript again.

## Auth

`gh` reads its config and token from `~/.config/gh/` and the system keyring —
user-level, so it's shared with any system-installed `gh` and survives
entering/leaving the devShell. No extra repo config needed.

```bash
gh auth status                # confirm gh is logged in, scope includes `repo`
gh auth refresh -s repo       # if the scope is missing
gs auth login                 # pick the "CLI" method — reuses gh's token
gs auth status
```

## One-time repo setup

```bash
just spice-init
```

Runs `gs repo init --trunk new-main --remote origin` and sets the repo-local
config below. git-spice's tracked-branch state lives in `refs/spice/data`,
shared across worktrees, so this only needs to run once per clone (not per
worktree).

## Daily loop

```bash
gs branch create <name>   # bc — commit staged changes to a new branch on the stack
gs commit create          # cc — new commit on the current branch
gs branch submit          # bs — push + open/update a PR for the current branch
gs stack submit           # ss — push + open/update PRs for the whole stack
gs up / gs down            # move between branches in the stack
gs branch restack          # r — rebase the current branch onto its updated base
gs log short               # ls — see the stack
```

Use `gs branch submit`, not `gh pr create` — GitHub's default branch is
`main`, so a bare `gh pr create` targets the wrong base. If you must use
`gh pr create` directly, pass `--base <parent-branch>`.

### Where `gh` still earns its place

`gs` handles branch/PR lifecycle; `gh` is still the tool for everything else:
`gh pr checks`, `gh pr view`, `gh pr diff`, `gh run watch`, and the existing
`gh co` alias (`pr checkout`) for checking out someone else's PR.

## Amending mid-stack

`gs commit amend` / `gs commit create` on a lower branch automatically
restacks the branches above it (`spice.commitAmend.restack` /
`spice.commitCreate.restack` default to `upstack`).

## Merging

The repo is squash-merge only with `delete_branch_on_merge`. Merge
bottom-up (`gs branch merge` on the lowest ready branch, or `gs stack merge`
for the whole stack), then:

```bash
gs repo sync
```

`gs repo sync` pulls `new-main`, detects branches whose PRs were squash-merged
and deletes them, and — because `repoSync.restack` is set to `upstack` —
restacks everything that was stacked above them. **Never rebase a stack by
hand**; `gs repo sync` is what reconciles a merged bottom branch.

## Conflicts

```bash
gs rebase continue   # after resolving conflicts during a restack
gs rebase abort
```

## Adopting existing branches

For branches created before git-spice was set up:

```bash
gs branch track --base new-main            # register a single branch
gs downstack track                         # register a branch and everything below it
```

This only registers the branch with git-spice; it doesn't rewrite history.

## Worktrees

git-spice's tracked-branch state is shared across worktrees via
`refs/spice/data` — init once, use from any worktree. Caveat:
`gs repo sync` won't delete a branch that's checked out in another worktree
even after it's merged (`spice.repoSync.detachWorktrees` would fix this but
isn't released yet as of git-spice 0.31.2).

## Config (`just spice-init`)

| Setting | Value | Why |
|---|---|---|
| `spice.submit.navigationComment` | `multiple` | Only post the stack-navigation comment when there are ≥2 PRs — no noise on solo PRs |
| `spice.repoSync.restack` | `upstack` | After a squash-merge, auto-restack the dependent branches |
| `spice.merge.method` | `squash` | Matches the repo's squash-only merge setting |
| `spice.log.pushStatusFormat` | `aheadBehind` | Shows `⇡1⇣2` in `gs log short` |
| `spice.branchPrompt.sort` | `-committerdate` | Recently touched branches first in interactive pickers |
| `spice.submit.web` | `created` | Opens a browser tab only for newly created PRs, not updates |

## Per-developer shell completion (optional)

Not shareable via the flake — direnv only exports environment variables, not
shell functions. Add to your own shell rc:

```bash
eval "$(gs shell completion zsh)"    # or bash/fish
eval "$(gh completion -s zsh)"
```

## Troubleshooting

- `gs: command not found` behaves like Ghostscript, or `gh --version` shows an
  old version → you're outside the devShell; run `direnv reload` or re-enter it.
- `gs log short --all` (`gs ls -a`) — see every stack, not just the current one.
- `gs auth status` / `gh auth status` — confirm both are logged in if PR
  operations fail.
