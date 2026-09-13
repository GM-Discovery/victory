# Migration Chronology

**Produced by:** Kernel 99, 2026-09-12. Maps all 114 migration files to the kernel that introduced them. Most migration filenames are already self-labeled with their originating kernel (`NNN_kernelXX_description.sql`) — this file makes that mapping explicit and flags the handful that aren't. Do not renumber any historical migration file; the runner (`internal/migrate`) sorts by filename and tracks a checksum ledger, not contiguous numbering, so gaps below are confirmed harmless rather than corruption.

---

## Migrations with no kernel attribution in their filename

| Migration | Kernel / Era | Purpose | Current status |
|---|---|---|---|
| `001_init.sql` | Pre-kernel-numbering (Era 1, Kernel 1) | Initial schema bootstrap | Active, foundational |
| `002_seed_world.sql` | Pre-kernel-numbering (Era 1, Kernel 1) | World/venue seed data | Active |
| `003_users_handle.sql` | Pre-kernel-numbering (Era 1, Kernel 1-2) | User handle column | Active |
| `004_seed_session.sql` | Pre-kernel-numbering (Era 1, Kernel 1) | Session seed data | Active |
| `013_element_context_class.sql` | UNKNOWN — no kernel citation found | Element context classification | Active, unattributed |
| `093_fix_missing_permission_requests_table.sql` | UNKNOWN — likely an out-of-band hotfix between Kernel 81 and 81A | Repairs a missing `permission_requests` table | Active; not cited by any Kernel 76-83 reportback per the existing Kernel 84 reconciliation ledger |

## Known numbering gap

**Migration `088` does not exist** — the sequence runs `087_kernel79a_ewrite_skill_directory.sql` directly to `089_kernel79_goalc_object_link_types.sql`. No reportback among Kernels 76-84 explains the gap. Confirmed harmless (per the existing `kernel-history-reconciliation-through-83.md`): the migration runner sorts by filename and tracks a checksum ledger, not contiguous numbers.

## Full chronology (all 114 migrations, kernel-attributed via filename)

| # | Migration | Kernel |
|---|---|---|
| 000 | `kernel42_productions_baseline.sql` | 42 |
| 001 | `init.sql` | — (pre-numbering) |
| 002 | `seed_world.sql` | — (pre-numbering) |
| 003 | `users_handle.sql` | — (pre-numbering) |
| 004 | `seed_session.sql` | — (pre-numbering) |
| 005 | `kernel2_identity_invites_assets.sql` | 2 |
| 006 | `kernel6_action_authority.sql` | 6 |
| 007 | `kernel9_profiles_greenroom_trailers.sql` | 9 |
| 008 | `kernel10_info_booth_mailbox.sql` | 10 |
| 009 | `kernel11_note_cards.sql` | 11 |
| 010 | `kernel13_workshop_placement.sql` | 13 |
| 011 | `kernel15_producer_director_offices.sql` | 15 |
| 012 | `kernel15_production_label.sql` | 15 |
| 013 | `element_context_class.sql` | — (unattributed) |
| 014 | `kernel21_venue_chat.sql` | 21 |
| 015 | `kernel22_showings.sql` | 22 |
| 016 | `kernel23_character_cards.sql` | 23 |
| 017 | `kernel24_character_sheet_links.sql` | 24 |
| 018 | `kernel32_discord_oauth.sql` | 32 |
| 019 | `kernel35_discord_server_link.sql` | 35 |
| 020 | `kernel35_discord_server_bootstrap.sql` | 35 |
| 021 | `kernel36_discord_channel_mappings.sql` | 36 |
| 022 | `kernel37_discord_mic_threads.sql` | 37 |
| 023 | `kernel38_discord_chat_bridges.sql` | 38 |
| 024 | `kernel39_discord_gateway_intake.sql` | 39 |
| 025 | `kernel42_neutral_install_location.sql` | 42 |
| 026 | `kernel46_first_theater_map.sql` | 46 |
| 027 | `kernel47_map_display_mode.sql` | 47 |
| 028 | `kernel47_grid_config.sql` | 47 |
| 029 | `kernel49_warehouse_storage.sql` | 49 |
| 030 | `kernel51_capacity_guardrails.sql` | 51 |
| 031 | `kernel53_character_workbook_foundation.sql` | 53 |
| 032 | `kernel59_command_registry.sql` | 59 |
| 033 | `kernel60_character_skills.sql` | 60 |
| 034 | `kernel59a_character_face_overrides.sql` | 59A |
| 035 | `kernel59a_director_value_overrides.sql` | 59A |
| 036 | `kernel61_player_workbook_foundation.sql` | 61 |
| 037 | `kernel62_player_relationships.sql` | 62 |
| 038 | `kernel65_third_place.sql` | 65 |
| 039 | `kernel66_show_runs.sql` | 66 |
| 040 | `kernel67_shows.sql` | 67 |
| 041 | `kernel68_productions_created_by.sql` | 68 |
| 042 | `kernel69_scenes.sql` | 69 |
| 043 | `kernel70_scene_location_scoping.sql` | 70 |
| 044 | `kernel70_show_stage_and_variables.sql` | 70 |
| 045 | `kernel70_cues.sql` | 70 |
| 046 | `kernel71_show_tickets.sql` | 71 |
| 047 | `kernel71_roster_character_selection.sql` | 71 |
| 048 | `kernel71_show_short_codes.sql` | 71 |
| 049 | `kernel72_profiles_surface_ddl.sql` | 72 |
| 050 | `kernel72_messages_surface_ddl.sql` | 72 |
| 051 | `kernel72_showings_surface_ddl.sql` | 72 |
| 052 | `kernel72_characters_surface_ddl.sql` | 72 |
| 053 | `kernel72_warehouse_surface_ddl.sql` | 72 |
| 054 | `kernel72_discord_gateway_surface_ddl.sql` | 72 |
| 055 | `kernel72a_stage_capability_flags.sql` | 72A |
| 056 | `kernel72a_index_cards_parity.sql` | 72A |
| 057 | `kernel73_courtyard_scene_seed.sql` | 73 |
| 058 | `kernel73_participant_interactions_capability.sql` | 73 |
| 059 | `kernel73_equipment_and_inventory.sql` | 73 |
| 060 | `kernel73_merchant_packets.sql` | 73 |
| 061 | `kernel73_participant_interactions.sql` | 73 |
| 062 | `kernel73_equipment_catalog_fields.sql` | 73 |
| 063 | `kernel73_socio_equipment_catalog_seed.sql` | 73 |
| 064 | `kernel73a_scene_stage_composition.sql` | 73A |
| 065 | `kernel73_open_enrollment.sql` | 73 |
| 066 | `kernel74_tutorial_progress_and_dialogue.sql` | 74 |
| 067 | `kernel74_local_projection_capability.sql` | 74 |
| 068 | `kernel74_projection_per_character.sql` | 74 |
| 069 | `kernel75_tutorial_completion_milestones.sql` | 75 |
| 070 | `kernel75_story_so_far.sql` | 75 |
| 071 | `kernel75_interaction_attempts.sql` | 75 |
| 072 | `kernel75_dialogue_authoring.sql` | 75 |
| 073 | `kernel75_aftercare.sql` | 75 |
| 074 | `kernel75_aftercare_capability.sql` | 75 |
| 075 | `kernel75_final_copy.sql` | 75 |
| 076 | `kernel75_director_journal.sql` | 75 |
| 077 | `kernel75_ra_opening_copy.sql` | 75 |
| 078 | `kernel75_ra_supervisor_handoff.sql` | 75 |
| 079 | `kernel75_ra_gate_reveal_compact.sql` | 75 |
| 080 | `kernel75_hide_handoff_card_for_players.sql` | 75 |
| 081 | `kernel77_deletion_tombstone.sql` | 77 |
| 082 | `kernel77_deletion_export_recovery.sql` | 77 |
| 083 | `kernel77a_audition_hall_seed_repair.sql` | 77A |
| 084 | `kernel78_ewrite_schema.sql` | 78 |
| 085 | `kernel78_writers_room_venue_seed.sql` | 78 |
| 086 | `kernel79_ewrite_publication_assets.sql` | 79 |
| 087 | `kernel79a_ewrite_skill_directory.sql` | 79A |
| *(088 does not exist — see above)* | | |
| 089 | `kernel79_goalc_object_link_types.sql` | 79 Goal C&E |
| 090 | `kernel80_storyboards_core.sql` | 80 |
| 091 | `kernel80_ewrite_object_link_storyboard_card.sql` | 80 |
| 092 | `kernel81_storyboard_card_image.sql` | 81 |
| 093 | `fix_missing_permission_requests_table.sql` | — (unattributed hotfix) |
| 094 | `kernel81a_storyboard_structural_slugs.sql` | 81A |
| 095 | `kernel82_storyboards_timeline_mode.sql` | 82 |
| 096 | `kernel85_cohorts.sql` | 85 |
| 097 | `kernel85_socio_mechanics.sql` | 85 |
| 098 | `kernel87_cartograph_drawing.sql` | 87 |
| 099 | `kernel87_cartograph_capability.sql` | 87 |
| 100 | `kernel88_socio_fate.sql` | 88 |
| 101 | `kernel88_socio_stance.sql` | 88 |
| 102 | `kernel88_socio_blank_flags.sql` | 88 |
| 103 | `kernel88_socio_interrupts.sql` | 88 |
| 104 | `kernel88_story_so_far_events.sql` | 88 |
| 105 | `kernel89_director_preparations.sql` | 89 |
| 106 | `kernel90_stage_object_state.sql` | 90 |
| 107 | `kernel91_tour_completions.sql` | 91 |
| 108 | `kernel91_tour_progress.sql` | 91 |
| 109 | `kernel92_showing_nickname.sql` | 92 |
| 110 | `kernel93_audience_admission.sql` | 93 |
| 111 | `kernel93_victory_theater_venue.sql` | 93 |
| 112 | `kernel93_tour_greenroom_split.sql` | 93 |
| 113 | `kernel93_mailbox_pin_delete.sql` | 93 |
| 114 | `kernel96_grants_cabin_contact.sql` | 96 |

**Migrations required for fresh install:** all 114 — the runner applies every migration in filename order on a fresh database with no historical skip logic found. This is proven directly by Part II's fresh-install proof rather than assumed here.
