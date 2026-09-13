# Operator

**What it is:** the installation's own infrastructure authority — the person who runs the Victory install itself, identified by configuration (`OPERATOR_USER_ID`/`OPERATOR_HANDLE`), not assigned as an in-product role. An Operator has full authority everywhere in Victory by construction.

**Home:** Grant's Cabin (or the equivalent Operator space in your install) for infrastructure-facing tools; otherwise the Operator moves through Victory like anyone else.

**Common actions:** manage the installation, recover accounts, view backstage state anywhere, act as Producer/Director anywhere without needing an explicit membership row.

**What it does not automatically mean:** being Operator does not silently change what role you *appear* as in the product — join a session as Audience, and you now genuinely see the Audience experience, not an automatically-upgraded one.

**Important privacy disclosure:** the server Operator may have technical access to the installation's database, files, backups, and administrative recovery tools. Application-level permissions protect users from unauthorized *other users* — they do not, and cannot, protect against the person who administers the machine the software runs on. This is ordinary self-hosted software reality, not a Victory-specific weakness, and it's worth knowing plainly if you're inviting people to your install.

**Contextual help:** Guide content appears in Grant's Cabin and wherever an infrastructure-facing action is available.
