-- Kernel 73 follow-up: the full Socio-: Stories of Us equipment catalog
-- as supplied by the operator, transcribed verbatim (cost/damage/tags/
-- effect text), seeded as reference data. Deactivates the 5 Kernel 73
-- placeholder items (kept, not deleted, so any existing purchase history
-- stays intact) and replaces them with a curated, narratively-sane
-- starter subset in Kessa's own shop -- the full catalog exists as
-- equipment_items rows for future merchants/kernels, but a fresh
-- Courtyard character cannot buy an Enchanted Plate or a War Horse from
-- Kessa specifically.

UPDATE equipment_items
SET active = FALSE
WHERE slug IN ('travelers-cloak', 'coil-of-rope', 'traveling-rations', 'sturdy-boots', 'simple-dagger')
  AND location_id = (SELECT id FROM locations WHERE is_default);

DELETE FROM merchant_packet_equipment_items
WHERE packet_id = (SELECT id FROM merchant_packets WHERE slug = 'kessa')
  AND equipment_item_id IN (
    SELECT id FROM equipment_items
    WHERE slug IN ('travelers-cloak', 'coil-of-rope', 'traveling-rations', 'sturdy-boots', 'simple-dagger')
      AND location_id = (SELECT id FROM locations WHERE is_default)
  );

INSERT INTO equipment_items (
  location_id, name, slug, category, cost_credits, short_description, quantity_mode, stats_json, active
)
SELECT l.id, v.name, v.slug, v.category, v.cost_credits, v.short_description, v.quantity_mode, v.stats_json::jsonb, TRUE
FROM locations l
JOIN (VALUES
  ('Loincloth', 'loincloth', 'clothing', 1, 'Minimal coverage.', 'unique', '{"effect": "-2 to social interactions in civilized areas"}'),
  ('Ragged Clothing', 'ragged-clothing', 'clothing', 5, 'Torn and patched.', 'unique', '{"effect": "-1 to social interactions"}'),
  ('Secondhand Clothing', 'secondhand-clothing', 'clothing', 10, 'Worn but serviceable.', 'unique', '{"effect": "No penalties"}'),
  ('Basic Clothing', 'basic-clothing', 'clothing', 20, 'New, clean, appropriate for most situations.', 'unique', '{}'),
  ('Fashionable Tier 1', 'fashionable-tier-1', 'clothing', 25, 'Stylish attire.', 'unique', '{"effect": "+1 Presence for bonuses, -1 TV on Presence attempts"}'),
  ('Fashionable Tier 2', 'fashionable-tier-2', 'clothing', 50, 'Stylish attire.', 'unique', '{"effect": "+1 Presence for bonuses, -2 TV on Presence attempts"}'),
  ('Fashionable Tier 3', 'fashionable-tier-3', 'clothing', 100, 'Stylish attire.', 'unique', '{"effect": "+1 Presence for bonuses, -3 TV on Presence attempts"}'),
  ('Fashionable Tier 4', 'fashionable-tier-4', 'clothing', 1000, 'Stylish attire.', 'unique', '{"effect": "+1 Presence for bonuses, -4 TV on Presence attempts"}'),
  ('Fashionable Tier 5', 'fashionable-tier-5', 'clothing', 10000, 'Stylish attire.', 'unique', '{"effect": "+1 Presence for bonuses, -5 TV on Presence attempts"}'),
  ('Traveling Cloak', 'traveling-cloak', 'clothing', 15, 'Weather protection.', 'unique', '{"effect": "+1 to Survival in poor weather"}'),
  ('Noble''s Signet Ring', 'nobles-signet-ring', 'clothing', 200, 'A mark of station.', 'unique', '{"effect": "Grants authority in specific domains"}'),
  ('Merchant''s Seal', 'merchants-seal', 'clothing', 50, 'A guild mark.', 'unique', '{"effect": "+2 to trade negotiations in relevant guilds"}'),
  ('Winter Furs', 'winter-furs', 'clothing', 80, 'Heavy cold-weather wear.', 'unique', '{"effect": "+3 to cold weather Survival, fashionable in northern regions"}'),
  ('Court Dress', 'court-dress', 'clothing', 150, 'Formal noble attire.', 'unique', '{"effect": "Required for formal noble functions, +2 to Etiquette"}'),
  ('Shared Floor', 'shelter-shared-floor', 'shelter', 1, 'A stable roof, limited privacy.', 'stackable', '{}'),
  ('Bed or Cot', 'shelter-bed-or-cot', 'shelter', 3, 'Recover HP overnight, remove minor statuses.', 'stackable', '{}'),
  ('Private Room', 'shelter-private-room', 'shelter', 8, 'Full recovery, narrative control, safe storage.', 'stackable', '{}'),
  ('Owned Home', 'shelter-owned-home', 'shelter', 500, 'Customize, base-building, long-term safety.', 'unique', '{"setup_cost": true}'),
  ('Scrap Meals', 'food-scrap-meals', 'food', 1, 'Fills the stomach, no healing.', 'stackable', '{}'),
  ('Stable Diet', 'food-stable-diet', 'food', 3, 'Heals minor HP loss with rest.', 'stackable', '{}'),
  ('Nourishing Meal', 'food-nourishing-meal', 'food', 6, 'Clears 1 status (e.g., Winded, Haunted).', 'stackable', '{}'),
  ('Shared Feast', 'food-shared-feast', 'food', 12, 'Group-wide morale bonus, Bond deepening.', 'stackable', '{}'),
  ('Cotton Padding', 'cotton-padding', 'armor', 8, 'Light, no movement penalty.', 'unique', '{"damage_reduction": 1}'),
  ('Leather Vest', 'leather-vest', 'armor', 25, 'Flexible.', 'unique', '{"damage_reduction": 2, "special": "+1 to Stealth"}'),
  ('Studded Leather', 'studded-leather', 'armor', 45, 'Reinforced, maintains flexibility.', 'unique', '{"damage_reduction": 3}'),
  ('Chain Shirt', 'chain-shirt', 'armor', 70, 'Metal links, moderate protection.', 'unique', '{"damage_reduction": 4}'),
  ('Scale Mail', 'scale-mail', 'armor', 90, 'Overlapping metal scales.', 'unique', '{"damage_reduction": 5}'),
  ('Chain Mail', 'chain-mail', 'armor', 120, 'Full chain coverage.', 'unique', '{"damage_reduction": 6, "special": "-1 to Grace actions"}'),
  ('Splint Armor', 'splint-armor', 'armor', 180, 'Metal strips on leather.', 'unique', '{"damage_reduction": 7}'),
  ('Banded Mail', 'banded-mail', 'armor', 250, 'Horizontal metal bands.', 'unique', '{"damage_reduction": 8}'),
  ('Plate Armor', 'plate-armor', 'armor', 500, 'Professional knight armor.', 'unique', '{"damage_reduction": 10, "special": "-2 to Grace actions"}'),
  ('Full Plate', 'full-plate', 'armor', 1200, 'Master crafted.', 'unique', '{"damage_reduction": 12, "special": "-3 to Grace actions"}'),
  ('Enchanted Plate', 'enchanted-plate', 'armor', 5000, 'Magical enhancement.', 'unique', '{"damage_reduction": 15, "special": "No Grace penalty"}'),
  ('Buckler', 'buckler', 'shield', 30, 'Parry bonus.', 'unique', '{"special": "+1 to Dodge, parry bonus"}'),
  ('Round Shield', 'round-shield', 'shield', 50, 'Can bash.', 'unique', '{"special": "+2 to Dodge, can bash"}'),
  ('Tower Shield', 'tower-shield', 'shield', 120, 'Provides cover.', 'unique', '{"special": "+3 to Dodge, provides cover"}'),
  ('Helmet, Basic', 'helmet-basic', 'armor', 25, 'Basic head protection.', 'unique', '{"damage_reduction": 1, "special": "head only"}'),
  ('Helmet, Great', 'helmet-great', 'armor', 80, 'Heavy head protection.', 'unique', '{"damage_reduction": 2, "special": "head only, vision penalty"}'),
  ('Knife', 'knife', 'weapon_blade', 5, 'Concealable, utility.', 'unique', '{"damage": "d4", "tags": ["Slashing", "Throwable"]}'),
  ('Dagger', 'dagger', 'weapon_blade', 12, 'Fast, precise.', 'unique', '{"damage": "d4", "tags": ["Slashing", "Piercing"]}'),
  ('Shortsword', 'shortsword', 'weapon_blade', 35, 'Light, versatile.', 'unique', '{"damage": "d6", "tags": ["Slashing", "One-Handed"]}'),
  ('Rapier', 'rapier', 'weapon_blade', 60, 'Precise, elegant.', 'unique', '{"damage": "d6", "tags": ["Piercing", "Finesse"]}'),
  ('Longsword', 'longsword', 'weapon_blade', 80, 'Balanced.', 'unique', '{"damage": "d8", "tags": ["Slashing", "Versatile"]}'),
  ('Scimitar', 'scimitar', 'weapon_blade', 70, 'Curved, swift.', 'unique', '{"damage": "d8", "tags": ["Slashing", "Swift"]}'),
  ('Bastard Sword', 'bastard-sword', 'weapon_blade', 120, 'One or two handed.', 'unique', '{"damage": "d10", "tags": ["Slashing", "Versatile"]}'),
  ('Greatsword', 'greatsword', 'weapon_blade', 180, 'Two-handed power.', 'unique', '{"damage": "d12", "tags": ["Slashing", "Two-Handed"]}'),
  ('Masterwork Blade', 'masterwork-blade', 'weapon_blade', 300, 'Superior craftsmanship (+300 over base blade).', 'unique', '{"damage": "d12+1d4", "tags": ["Masterwork"]}'),
  ('Club', 'club', 'weapon_blunt', 2, 'Simple.', 'unique', '{"damage": "d4", "tags": ["Crushing", "Improvised"]}'),
  ('Mace', 'mace', 'weapon_blunt', 25, 'Spiked head.', 'unique', '{"damage": "d6", "tags": ["Crushing", "Anti-Armor"]}'),
  ('War Hammer', 'war-hammer', 'weapon_blunt', 45, 'Armor crushing.', 'unique', '{"damage": "d8", "tags": ["Crushing", "Two-Handed"]}'),
  ('Maul', 'maul', 'weapon_blunt', 70, 'Heavy crusher.', 'unique', '{"damage": "d10", "tags": ["Crushing", "Two-Handed"]}'),
  ('Flail', 'flail', 'weapon_blunt', 55, 'Chain weapon.', 'unique', '{"damage": "d8", "tags": ["Crushing", "Unpredictable"]}'),
  ('Morning Star', 'morning-star', 'weapon_blunt', 40, 'Spiked sphere.', 'unique', '{"damage": "d6", "tags": ["Crushing", "Piercing"]}'),
  ('Spear', 'spear', 'weapon_polearm', 20, 'Reach, throwable.', 'unique', '{"damage": "d6", "tags": ["Piercing", "Reach"]}'),
  ('Pike', 'pike', 'weapon_polearm', 35, 'Anti-cavalry.', 'unique', '{"damage": "d8", "tags": ["Piercing", "Long Reach"]}'),
  ('Halberd', 'halberd', 'weapon_polearm', 90, 'Axe-spear hybrid.', 'unique', '{"damage": "d10", "tags": ["Slashing", "Piercing", "Reach"]}'),
  ('Glaive', 'glaive', 'weapon_polearm', 75, 'Blade on pole.', 'unique', '{"damage": "d8", "tags": ["Slashing", "Reach"]}'),
  ('Partisan', 'partisan', 'weapon_polearm', 85, 'Decorated spear.', 'unique', '{"damage": "d8", "tags": ["Piercing", "Reach", "Noble"]}'),
  ('Sling', 'sling', 'weapon_ranged', 3, 'Simple, silent.', 'unique', '{"damage": "d4", "tags": ["Ranged", "Stealth"]}'),
  ('Shortbow', 'shortbow', 'weapon_ranged', 40, 'Fast shooting.', 'unique', '{"damage": "d6", "tags": ["Ranged", "Quick Draw"]}'),
  ('Longbow', 'longbow', 'weapon_ranged', 80, 'Long range.', 'unique', '{"damage": "d8", "tags": ["Ranged", "Long Range"]}'),
  ('Crossbow', 'crossbow', 'weapon_ranged', 120, 'Mechanical.', 'unique', '{"damage": "d10", "tags": ["Ranged", "Armor Piercing"]}'),
  ('Heavy Crossbow', 'heavy-crossbow', 'weapon_ranged', 200, 'Siege weapon.', 'unique', '{"damage": "d12", "tags": ["Ranged", "Two-Handed"]}'),
  ('Throwing Knife', 'throwing-knife', 'weapon_ranged', 8, 'Concealable.', 'unique', '{"damage": "d4", "tags": ["Thrown", "Stealth"]}'),
  ('Javelin', 'javelin', 'weapon_ranged', 15, 'Thrown spear.', 'unique', '{"damage": "d6", "tags": ["Thrown", "Piercing"]}'),
  ('Arrows (20)', 'arrows-20', 'weapon_ranged', 5, 'Ammunition.', 'stackable', '{"tags": ["Consumable"]}'),
  ('Bolts (20)', 'bolts-20', 'weapon_ranged', 8, 'Ammunition.', 'stackable', '{"tags": ["Consumable"]}'),
  ('Artisan''s Tools', 'artisans-tools', 'tool', 50, 'Required for Craft skills.', 'unique', '{"effect": "+1 to Craft skills"}'),
  ('Smith''s Hammer', 'smiths-hammer', 'tool', 25, 'A smith''s implement.', 'unique', '{"effect": "+1 to metalwork"}'),
  ('Thieves'' Tools', 'thieves-tools', 'tool', 80, 'Lockpicks and picks.', 'unique', '{"effect": "+2 to lockpicking"}'),
  ('Healer''s Kit', 'healers-kit', 'tool', 45, '10 uses.', 'unique', '{"effect": "Medical treatment, 10 uses"}'),
  ('Scholar''s Kit', 'scholars-kit', 'tool', 60, 'Research materials.', 'unique', '{"effect": "+1 to Research"}'),
  ('Climbing Gear', 'climbing-gear', 'tool', 35, 'Pitons and line.', 'unique', '{"effect": "+2 to Climbing"}'),
  ('Survival Kit', 'survival-kit', 'tool', 40, 'General wilderness gear.', 'unique', '{"effect": "+1 to Survival"}'),
  ('Musical Instrument', 'musical-instrument', 'tool', 30, 'Required for Performance.', 'unique', '{"effect": "Required for Performance"}'),
  ('Magnifying Glass', 'magnifying-glass', 'tool', 80, 'Fine detail work.', 'unique', '{"effect": "+2 to Search"}'),
  ('Telescope', 'telescope', 'tool', 150, 'Long-distance viewing.', 'unique', '{"effect": "+3 to Observation"}'),
  ('Backpack', 'backpack', 'adventuring_supply', 8, 'Carry more.', 'unique', '{}'),
  ('Bedroll', 'bedroll', 'adventuring_supply', 3, 'Rest comfort.', 'unique', '{}'),
  ('Rope (50 ft)', 'rope-50ft', 'adventuring_supply', 10, 'Utility rope.', 'unique', '{}'),
  ('Silk Rope (50 ft)', 'silk-rope-50ft', 'adventuring_supply', 50, 'Quiet, light.', 'unique', '{}'),
  ('Torch', 'torch', 'adventuring_supply', 1, '1 hour of light.', 'stackable', '{}'),
  ('Lantern', 'lantern', 'adventuring_supply', 15, 'Requires oil.', 'unique', '{}'),
  ('Oil (1 flask)', 'oil-flask', 'adventuring_supply', 2, 'Fuel.', 'stackable', '{}'),
  ('Rations (1 day)', 'rations-1day', 'adventuring_supply', 2, 'Sustenance.', 'stackable', '{}'),
  ('Waterskin', 'waterskin', 'adventuring_supply', 3, '1 day of water.', 'unique', '{}'),
  ('Tent, Small', 'tent-small', 'adventuring_supply', 25, '2 people.', 'unique', '{}'),
  ('Tent, Large', 'tent-large', 'adventuring_supply', 60, '6 people.', 'unique', '{}'),
  ('Winter Blanket', 'winter-blanket', 'adventuring_supply', 12, 'Cold protection.', 'unique', '{"effect": "+1 cold Survival"}'),
  ('Spyglass', 'spyglass', 'luxury', 200, 'For seafaring.', 'unique', '{"effect": "+4 sea navigation"}'),
  ('Silk Clothing', 'silk-clothing', 'luxury', 100, 'Exotic fabric.', 'unique', '{"effect": "+1 exotic interactions"}'),
  ('Fine Jewelry', 'fine-jewelry', 'luxury', 500, 'A display of wealth.', 'unique', '{"effect": "+2 wealth displays"}'),
  ('Exotic Perfume', 'exotic-perfume', 'luxury', 80, 'A memorable scent.', 'unique', '{"effect": "+1 first impressions"}'),
  ('Masterwork Instrument', 'masterwork-instrument', 'luxury', 400, 'Superb craftsmanship.', 'unique', '{"effect": "+1d4 Performance"}'),
  ('Blessed Holy Symbol', 'blessed-holy-symbol', 'luxury', 150, 'A consecrated icon.', 'unique', '{"effect": "+1 Faith"}'),
  ('Spell Component Pouch', 'spell-component-pouch', 'luxury', 120, 'Required for magic.', 'unique', '{"effect": "Required for magic"}'),
  ('Crystal Ball', 'crystal-ball', 'luxury', 800, 'A scrying tool.', 'unique', '{"effect": "+3 divination"}'),
  ('Magic Scroll', 'magic-scroll', 'luxury', 300, 'Single-use.', 'unique', '{"effect": "Single-use magic"}'),
  ('Potion of Healing', 'potion-of-healing', 'luxury', 50, 'Restores HP.', 'stackable', '{"effect": "Restore 1d6 HP"}'),
  ('Riding Horse', 'riding-horse', 'mount', 200, 'A reliable mount.', 'unique', '{"effect": "+3 travel"}'),
  ('War Horse', 'war-horse', 'mount', 800, 'Trained for battle.', 'unique', '{"effect": "+2 mounted combat"}'),
  ('Pony', 'pony', 'mount', 100, 'Sure-footed.', 'unique', '{"effect": "+2 mountain travel"}'),
  ('Mule', 'mule', 'mount', 80, 'A pack animal.', 'unique', '{"effect": "Heavy loads"}'),
  ('Ox', 'ox', 'mount', 120, 'A draft animal.', 'unique', '{"effect": "Farming"}'),
  ('Hunting Dog', 'hunting-dog', 'mount', 60, 'A trained tracker.', 'unique', '{"effect": "+2 Tracking"}'),
  ('Guard Dog', 'guard-dog', 'mount', 40, 'A watchful companion.', 'unique', '{"effect": "+1 Vigilance"}'),
  ('Falcon', 'falcon', 'mount', 150, 'A trained messenger bird.', 'unique', '{"effect": "Messenger"}'),
  ('Inn Stay (night)', 'inn-stay-night', 'service', 2, 'Room and a meal.', 'stackable', '{}'),
  ('Tavern Meal', 'tavern-meal', 'service', 1, 'A social meal.', 'stackable', '{}'),
  ('Stable Mount', 'stable-mount', 'service', 1, 'Safekeeping for a mount.', 'stackable', '{}'),
  ('Hire Guide', 'hire-guide', 'service', 10, 'Per day.', 'stackable', '{"effect": "+3 Navigation"}'),
  ('Messenger', 'messenger', 'service', 5, 'City delivery.', 'stackable', '{}'),
  ('Scholar Consultation', 'scholar-consultation', 'service', 20, 'Expert advice.', 'stackable', '{"effect": "+2 Lore"}'),
  ('Map, Local', 'map-local', 'service', 15, 'A local map.', 'unique', '{"effect": "+1 Navigation"}'),
  ('Map, Regional', 'map-regional', 'service', 50, 'A regional map.', 'unique', '{"effect": "+2 Navigation"}'),
  ('Rumors', 'rumors', 'service', 5, 'Local gossip and leads (variable cost, 5-50).', 'stackable', '{}'),
  ('Safe Passage Papers', 'safe-passage-papers', 'service', 100, 'Legal travel documents.', 'unique', '{}'),
  ('Ring of Protection', 'ring-of-protection', 'magic_item', 2000, 'A warded band.', 'unique', '{"effect": "+1 defense"}'),
  ('Cloak of Elvenkind', 'cloak-of-elvenkind', 'magic_item', 1500, 'A concealing cloak.', 'unique', '{"effect": "+3 Stealth"}'),
  ('Boots of Speed', 'boots-of-speed', 'magic_item', 1800, 'Enchanted footwear.', 'unique', '{"effect": "+2 movement"}'),
  ('Bag of Holding', 'bag-of-holding', 'magic_item', 3000, 'A spatial pocket.', 'unique', '{"effect": "Unlimited carry"}'),
  ('Wand of Magic Missiles', 'wand-of-magic-missiles', 'magic_item', 2500, '20 charges.', 'unique', '{"effect": "20 charges"}'),
  ('Greater Healing Potion', 'greater-healing-potion', 'magic_item', 150, 'A potent draught.', 'stackable', '{"effect": "Restore 2d6 HP"}'),
  ('Potion of Strength', 'potion-of-strength', 'magic_item', 100, 'A muscle-bound draught.', 'stackable', '{"effect": "+2 Might"}'),
  ('Scroll of Fireball', 'scroll-of-fireball', 'magic_item', 200, 'Single-use area damage.', 'stackable', '{"effect": "Area damage"}'),
  ('Enchanted Weapon +1', 'enchanted-weapon-plus-1', 'magic_item', 2000, 'A magically sharpened edge (+2000 over base weapon).', 'unique', '{"effect": "+1d4 damage"}'),
  ('Amulet of Truth', 'amulet-of-truth', 'magic_item', 1200, 'Reveals falsehood.', 'unique', '{"effect": "+3 lie detection"}'),
  ('Basic Poison', 'basic-poison', 'contraband', 80, 'Illicit toxin.', 'stackable', '{"effect": "Ongoing damage"}'),
  ('Lethal Poison', 'lethal-poison', 'contraband', 300, 'A deadly toxin.', 'stackable', '{"effect": "Potentially fatal"}'),
  ('Forgery Kit', 'forgery-kit', 'contraband', 150, 'Documents and inks.', 'unique', '{"effect": "+2 forgery"}'),
  ('Disguise Kit', 'disguise-kit', 'contraband', 100, 'Makeup and props.', 'unique', '{"effect": "+2 disguise"}'),
  ('Smoke Bomb', 'smoke-bomb', 'contraband', 20, 'A concealment device.', 'stackable', '{"effect": "Concealment"}'),
  ('Flash Powder', 'flash-powder', 'contraband', 25, 'A blinding charge.', 'stackable', '{"effect": "Blind 1 round"}'),
  ('Master Lock Picks', 'master-lock-picks', 'contraband', 200, 'Fine picks.', 'unique', '{"effect": "+3 lockpicking"}'),
  ('False Identity Papers', 'false-identity-papers', 'contraband', 500, 'A new identity.', 'unique', '{"effect": "New identity"}'),
  ('Fresh Bread', 'fresh-bread', 'consumable', 0.5, 'Simple sustenance.', 'stackable', '{}'),
  ('Cheese Wheel', 'cheese-wheel', 'consumable', 8, 'Long-lasting.', 'stackable', '{}'),
  ('Smoked Meat', 'smoked-meat', 'consumable', 12, 'Preserved meat.', 'stackable', '{}'),
  ('Common Wine', 'common-wine', 'consumable', 3, 'A casual drink.', 'stackable', '{"effect": "+1 casual social"}'),
  ('Fine Wine', 'fine-wine', 'consumable', 25, 'A refined vintage.', 'stackable', '{"effect": "+2 formal"}'),
  ('Ale (pint)', 'ale-pint', 'consumable', 0.5, 'A tavern staple.', 'stackable', '{}'),
  ('Strong Spirits', 'strong-spirits', 'consumable', 15, 'Potent liquor.', 'stackable', '{"effect": "+1 Intimidation, -1 Precision"}'),
  ('Exotic Spices', 'exotic-spices', 'consumable', 50, 'Rare seasoning.', 'stackable', '{"effect": "+1 Cooking"}'),
  ('Medicinal Tea', 'medicinal-tea', 'consumable', 10, 'A soothing brew.', 'stackable', '{"effect": "+1 recovery"}'),
  ('Coffee', 'coffee', 'consumable', 20, 'A bracing drink.', 'stackable', '{"effect": "+1 alertness"}')
) AS v(name, slug, category, cost_credits, short_description, quantity_mode, stats_json) ON TRUE
WHERE l.is_default
  AND NOT EXISTS (
    SELECT 1 FROM equipment_items e WHERE e.location_id = l.id AND e.slug = v.slug
  );

-- Kessa's curated starter stock -- a fresh Courtyard character's sane
-- shopping list, not the full catalog. Operator-approved subset.
INSERT INTO merchant_packet_equipment_items (packet_id, equipment_item_id, sort_order)
SELECT mp.id, ei.id, v.ord
FROM merchant_packets mp
JOIN locations l ON l.id = mp.location_id AND l.is_default
JOIN (VALUES
  ('secondhand-clothing', 1),
  ('traveling-cloak', 2),
  ('rations-1day', 3),
  ('waterskin', 4),
  ('tent-small', 5),
  ('bedroll', 6),
  ('torch', 7),
  ('rope-50ft', 8),
  ('backpack', 9),
  ('knife', 10),
  ('dagger', 11),
  ('club', 12),
  ('spear', 13),
  ('sling', 14),
  ('shortbow', 15),
  ('cotton-padding', 16),
  ('leather-vest', 17),
  ('buckler', 18),
  ('survival-kit', 19)
) AS v(slug, ord) ON TRUE
JOIN equipment_items ei ON ei.location_id = l.id AND ei.slug = v.slug
WHERE mp.slug = 'kessa'
  AND NOT EXISTS (
    SELECT 1 FROM merchant_packet_equipment_items x
    WHERE x.packet_id = mp.id AND x.equipment_item_id = ei.id
  );
