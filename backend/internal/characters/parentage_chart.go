package characters

import (
	"fmt"
	"strings"
)

const ParentageChartVersionV11 = "1.1"

const ParentageOrganizationalCreditFloor = 300000

type ParentageChartEntry struct {
	RollMin        int    `json:"roll_min"`
	RollMax        int    `json:"roll_max"`
	SocialClass    string `json:"social_class"`
	WealthKind     string `json:"wealth_kind"`
	StartingCredit int    `json:"starting_credit"`
	Description    string `json:"description"`
}

var ParentageChartV11 = []ParentageChartEntry{
	{RollMin: 3, RollMax: 3, SocialClass: "Abandoned", WealthKind: "personal", StartingCredit: 0, Description: "No known parents, raised by streets/charity"},
	{RollMin: 4, RollMax: 4, SocialClass: "Orphaned", WealthKind: "personal", StartingCredit: 0, Description: "Parents died, no inheritance, institutional care"},
	{RollMin: 5, RollMax: 5, SocialClass: "Foundling", WealthKind: "personal", StartingCredit: 0, Description: "Left at doorstep, monastery, or temple"},
	{RollMin: 6, RollMax: 6, SocialClass: "Runaway Ward", WealthKind: "personal", StartingCredit: 0, Description: "Escaped poor guardianship, self-reliant"},
	{RollMin: 7, RollMax: 7, SocialClass: "Street Child", WealthKind: "personal", StartingCredit: 0, Description: "Unhoused from birth, scavenged survival"},
	{RollMin: 8, RollMax: 8, SocialClass: "Workhouse Orphan", WealthKind: "personal", StartingCredit: 1, Description: "Institutional labor, minimal possessions"},
	{RollMin: 9, RollMax: 9, SocialClass: "Refugee Child", WealthKind: "personal", StartingCredit: 1, Description: "War/disaster displaced, lost everything"},
	{RollMin: 10, RollMax: 10, SocialClass: "Feral Child", WealthKind: "personal", StartingCredit: 1, Description: "Raised by animals/nature, no human society"},
	{RollMin: 11, RollMax: 11, SocialClass: "Beggar's Child", WealthKind: "personal", StartingCredit: 2, Description: "Parent homeless, subsistence begging"},
	{RollMin: 12, RollMax: 12, SocialClass: "Scavenger Family", WealthKind: "personal", StartingCredit: 3, Description: "Survive on refuse, scraps, urban foraging"},
	{RollMin: 13, RollMax: 13, SocialClass: "Indentured Offspring", WealthKind: "personal", StartingCredit: 4, Description: "Parent endured binding agreements for survival"},
	{RollMin: 14, RollMax: 14, SocialClass: "Outcast Lineage", WealthKind: "personal", StartingCredit: 5, Description: "Family exiled, social pariahs, shunned"},
	{RollMin: 15, RollMax: 15, SocialClass: "Abandoned Heir", WealthKind: "personal", StartingCredit: 6, Description: "Disowned at birth, fallen from birthrights"},
	{RollMin: 16, RollMax: 16, SocialClass: "Criminal's Child", WealthKind: "personal", StartingCredit: 7, Description: "Parent imprisoned, family stigmatized"},
	{RollMin: 17, RollMax: 17, SocialClass: "Nomad Wanderer", WealthKind: "personal", StartingCredit: 8, Description: "Rootless travelers, no permanent settlement"},
	{RollMin: 18, RollMax: 18, SocialClass: "Subsistence Fisher", WealthKind: "personal", StartingCredit: 9, Description: "Coastal/river survival, minimal catch income"},
	{RollMin: 19, RollMax: 19, SocialClass: "Seasonal Laborer", WealthKind: "personal", StartingCredit: 10, Description: "Harvest work, inconsistent employment"},
	{RollMin: 20, RollMax: 20, SocialClass: "Rag Picker", WealthKind: "personal", StartingCredit: 12, Description: "Urban waste collection, textile salvage"},
	{RollMin: 21, RollMax: 21, SocialClass: "Street Sweeper", WealthKind: "personal", StartingCredit: 15, Description: "Municipal sanitation, minimal wage"},
	{RollMin: 22, RollMax: 22, SocialClass: "Tavern Keeper/Stable Hand", WealthKind: "personal", StartingCredit: 18, Description: "Service industry, tips and lodging"},
	{RollMin: 23, RollMax: 23, SocialClass: "Day Laborer", WealthKind: "personal", StartingCredit: 20, Description: "Construction/dock work, daily wages"},
	{RollMin: 24, RollMax: 24, SocialClass: "Apprentice Worker", WealthKind: "personal", StartingCredit: 25, Description: "Learning trade, minimal compensation"},
	{RollMin: 25, RollMax: 25, SocialClass: "Mill Worker", WealthKind: "personal", StartingCredit: 28, Description: "Factory labor, regular but low wages"},
	{RollMin: 26, RollMax: 26, SocialClass: "Farm Hand", WealthKind: "personal", StartingCredit: 30, Description: "Agricultural labor, seasonal employment"},
	{RollMin: 27, RollMax: 27, SocialClass: "Kitchen Staff", WealthKind: "personal", StartingCredit: 32, Description: "Restaurant/manor cooking, steady work"},
	{RollMin: 28, RollMax: 28, SocialClass: "Laundry Worker", WealthKind: "personal", StartingCredit: 35, Description: "Cleaning services, modest income"},
	{RollMin: 29, RollMax: 29, SocialClass: "Market Vendor", WealthKind: "personal", StartingCredit: 38, Description: "Small goods sales, variable income"},
	{RollMin: 30, RollMax: 30, SocialClass: "Courier/Messenger", WealthKind: "personal", StartingCredit: 40, Description: "Urban delivery, foot transportation"},
	{RollMin: 31, RollMax: 31, SocialClass: "Night Watchperson", WealthKind: "personal", StartingCredit: 45, Description: "Security guard, steady night work"},
	{RollMin: 32, RollMax: 32, SocialClass: "Seamstress/Tailor", WealthKind: "personal", StartingCredit: 50, Description: "Clothing repair, skilled needle work"},
	{RollMin: 33, RollMax: 33, SocialClass: "Baker's Assistant", WealthKind: "personal", StartingCredit: 50, Description: "Food production, early morning shifts"},
	{RollMin: 34, RollMax: 34, SocialClass: "Blacksmith's Helper", WealthKind: "personal", StartingCredit: 52, Description: "Metalworking aid, physical labor"},
	{RollMin: 35, RollMax: 35, SocialClass: "Carpenter's Apprentice", WealthKind: "personal", StartingCredit: 55, Description: "Woodworking training, tool learning"},
	{RollMin: 36, RollMax: 36, SocialClass: "Merchant's Clerk", WealthKind: "personal", StartingCredit: 60, Description: "Bookkeeping, inventory management"},
	{RollMin: 37, RollMax: 37, SocialClass: "Inn Keeper", WealthKind: "personal", StartingCredit: 65, Description: "Hospitality management, room rentals"},
	{RollMin: 38, RollMax: 38, SocialClass: "Skilled Artisan", WealthKind: "personal", StartingCredit: 70, Description: "Craft specialization, custom work"},
	{RollMin: 39, RollMax: 39, SocialClass: "Guild Member", WealthKind: "personal", StartingCredit: 75, Description: "Professional association, protected trade"},
	{RollMin: 40, RollMax: 40, SocialClass: "Small Shop Owner", WealthKind: "personal", StartingCredit: 80, Description: "Independent business, local clientele"},
	{RollMin: 41, RollMax: 41, SocialClass: "Regional Trader", WealthKind: "personal", StartingCredit: 85, Description: "Multi-town commerce, wagon routes"},
	{RollMin: 42, RollMax: 42, SocialClass: "Master Craftsman", WealthKind: "personal", StartingCredit: 100, Description: "Expert artisan, sought-after work"},
	{RollMin: 43, RollMax: 43, SocialClass: "Successful Merchant", WealthKind: "personal", StartingCredit: 110, Description: "Established trade routes, good profits"},
	{RollMin: 44, RollMax: 44, SocialClass: "Ship Captain", WealthKind: "personal", StartingCredit: 120, Description: "Maritime commerce, vessel ownership"},
	{RollMin: 45, RollMax: 45, SocialClass: "Warehouse Owner", WealthKind: "personal", StartingCredit: 130, Description: "Storage facility, distribution hub"},
	{RollMin: 46, RollMax: 46, SocialClass: "Manufacturing Owner", WealthKind: "personal", StartingCredit: 140, Description: "Small factory/workshop, employees"},
	{RollMin: 47, RollMax: 47, SocialClass: "Professional", WealthKind: "personal", StartingCredit: 150, Description: "Doctor/Lawyer/Paramedic, educated service provider"},
	{RollMin: 48, RollMax: 48, SocialClass: "Guild Master", WealthKind: "personal", StartingCredit: 160, Description: "Trade organization leader, influence"},
	{RollMin: 49, RollMax: 49, SocialClass: "Tavern/Inn Chain Owner", WealthKind: "personal", StartingCredit: 170, Description: "Multiple hospitality properties"},
	{RollMin: 50, RollMax: 50, SocialClass: "Regional Distributor", WealthKind: "personal", StartingCredit: 180, Description: "Large-scale goods movement"},
	{RollMin: 51, RollMax: 51, SocialClass: "Shipping Company", WealthKind: "personal", StartingCredit: 200, Description: "Fleet owner, maritime empire"},
	{RollMin: 52, RollMax: 52, SocialClass: "Established Merchant House", WealthKind: "personal", StartingCredit: 220, Description: "Family business, generational wealth"},
	{RollMin: 53, RollMax: 53, SocialClass: "Import/Export Specialist", WealthKind: "personal", StartingCredit: 240, Description: "International trade, exotic goods"},
	{RollMin: 54, RollMax: 54, SocialClass: "Banking Associate", WealthKind: "personal", StartingCredit: 260, Description: "Financial services, lending"},
	{RollMin: 55, RollMax: 55, SocialClass: "Property Developer", WealthKind: "personal", StartingCredit: 280, Description: "Real estate, construction projects"},
	{RollMin: 56, RollMax: 56, SocialClass: "Mining Operations", WealthKind: "personal", StartingCredit: 300, Description: "Resource extraction, mineral rights"},
	{RollMin: 57, RollMax: 57, SocialClass: "Transportation Magnate", WealthKind: "personal", StartingCredit: 320, Description: "Multiple shipping/cargo services"},
	{RollMin: 58, RollMax: 58, SocialClass: "Manufacturing Consortium", WealthKind: "personal", StartingCredit: 340, Description: "Industrial production, market share"},
	{RollMin: 59, RollMax: 59, SocialClass: "Agricultural Estate", WealthKind: "personal", StartingCredit: 360, Description: "Large farms, tenant farmers"},
	{RollMin: 60, RollMax: 60, SocialClass: "Merchant Banking", WealthKind: "personal", StartingCredit: 380, Description: "Commercial lending, investment"},
	{RollMin: 61, RollMax: 61, SocialClass: "Trade Route Controller", WealthKind: "personal", StartingCredit: 400, Description: "Monopoly over specific routes"},
	{RollMin: 62, RollMax: 62, SocialClass: "Wealthy Professional", WealthKind: "personal", StartingCredit: 450, Description: "Highly paid specialist, reputation"},
	{RollMin: 63, RollMax: 63, SocialClass: "Regional Bank Owner", WealthKind: "personal", StartingCredit: 500, Description: "Financial institution, local power"},
	{RollMin: 64, RollMax: 64, SocialClass: "Industrial Investor", WealthKind: "personal", StartingCredit: 550, Description: "Factory ownership, passive income"},
	{RollMin: 65, RollMax: 65, SocialClass: "Luxury Goods Merchant", WealthKind: "personal", StartingCredit: 600, Description: "High-end clientele, premium products"},
	{RollMin: 66, RollMax: 66, SocialClass: "Estate Manager", WealthKind: "personal", StartingCredit: 650, Description: "Noble property administrator"},
	{RollMin: 67, RollMax: 67, SocialClass: "Commercial Real Estate", WealthKind: "personal", StartingCredit: 700, Description: "Business district properties"},
	{RollMin: 68, RollMax: 68, SocialClass: "Shipping Fleet Owner", WealthKind: "personal", StartingCredit: 750, Description: "Multiple vessels, trade networks"},
	{RollMin: 69, RollMax: 69, SocialClass: "Manufacturing Empire", WealthKind: "personal", StartingCredit: 800, Description: "Industrial production, employees"},
	{RollMin: 70, RollMax: 70, SocialClass: "Financial Investor", WealthKind: "personal", StartingCredit: 850, Description: "Stock market, diverse portfolio"},
	{RollMin: 71, RollMax: 71, SocialClass: "Resource Monopolist", WealthKind: "personal", StartingCredit: 1000, Description: "Exclusive access, market control"},
	{RollMin: 72, RollMax: 72, SocialClass: "Merchant Prince", WealthKind: "personal", StartingCredit: 1200, Description: "Trade empire, political influence"},
	{RollMin: 73, RollMax: 73, SocialClass: "Minor Noble", WealthKind: "personal", StartingCredit: 1400, Description: "Hereditary title, land holdings"},
	{RollMin: 74, RollMax: 74, SocialClass: "Banking Dynasty", WealthKind: "personal", StartingCredit: 1600, Description: "Multi-generational finance house"},
	{RollMin: 75, RollMax: 75, SocialClass: "Industrial Baron", WealthKind: "personal", StartingCredit: 1800, Description: "Manufacturing empire, innovation"},
	{RollMin: 76, RollMax: 76, SocialClass: "Shipping Magnate", WealthKind: "personal", StartingCredit: 2000, Description: "International fleet, global reach"},
	{RollMin: 77, RollMax: 77, SocialClass: "Land Baron", WealthKind: "personal", StartingCredit: 2500, Description: "Vast estates, agricultural wealth"},
	{RollMin: 78, RollMax: 78, SocialClass: "Political Powerbroker", WealthKind: "personal", StartingCredit: 3000, Description: "Government influence, connections"},
	{RollMin: 79, RollMax: 79, SocialClass: "Financial Empire", WealthKind: "personal", StartingCredit: 3500, Description: "Multiple banks, investment houses"},
	{RollMin: 80, RollMax: 80, SocialClass: "Trade Monopolist", WealthKind: "personal", StartingCredit: 4000, Description: "Exclusive rights, market domination"},
	{RollMin: 81, RollMax: 81, SocialClass: "Industrial Titan", WealthKind: "personal", StartingCredit: 5000, Description: "Factory cities, worker populations"},
	{RollMin: 82, RollMax: 82, SocialClass: "Merchant Emperor", WealthKind: "personal", StartingCredit: 6000, Description: "Continental trade, economic power"},
	{RollMin: 83, RollMax: 83, SocialClass: "Lesser Nobility", WealthKind: "personal", StartingCredit: 8000, Description: "Ancient bloodline, court position"},
	{RollMin: 84, RollMax: 84, SocialClass: "Ducal Family", WealthKind: "personal", StartingCredit: 10000, Description: "Regional governance, military command"},
	{RollMin: 85, RollMax: 85, SocialClass: "Royal Cousin", WealthKind: "personal", StartingCredit: 12000, Description: "Distant crown relation, palace access"},
	{RollMin: 86, RollMax: 86, SocialClass: "Merchant Royalty", WealthKind: "personal", StartingCredit: 15000, Description: "Commercial empire, urban kingdoms"},
	{RollMin: 87, RollMax: 87, SocialClass: "Economic Oligarch", WealthKind: "personal", StartingCredit: 18000, Description: "Market manipulation, price control"},
	{RollMin: 88, RollMax: 88, SocialClass: "Industrial Dynasty", WealthKind: "personal", StartingCredit: 22000, Description: "Generational manufacturing empire"},
	{RollMin: 89, RollMax: 89, SocialClass: "Banking Royalty", WealthKind: "personal", StartingCredit: 26000, Description: "Financial kingdoms, currency influence"},
	{RollMin: 90, RollMax: 90, SocialClass: "Commercial Emperor", WealthKind: "personal", StartingCredit: 30000, Description: "Trade empire spanning continents"},
	{RollMin: 91, RollMax: 91, SocialClass: "Resource Sovereign", WealthKind: "personal", StartingCredit: 35000, Description: "Control of entire industries"},
	{RollMin: 92, RollMax: 92, SocialClass: "Established Aristocracy", WealthKind: "personal", StartingCredit: 50000, Description: "Ancient noble house, vast holdings"},
	{RollMin: 93, RollMax: 93, SocialClass: "Royal Treasury", WealthKind: "personal", StartingCredit: 75000, Description: "Crown wealth, national resources"},
	{RollMin: 94, RollMax: 94, SocialClass: "Economic Superpower", WealthKind: "personal", StartingCredit: 100000, Description: "Multi-national influence"},
	{RollMin: 95, RollMax: 95, SocialClass: "Industrial Empire", WealthKind: "personal", StartingCredit: 150000, Description: "Manufacturing across nations"},
	{RollMin: 96, RollMax: 96, SocialClass: "Financial Colossus", WealthKind: "personal", StartingCredit: 200000, Description: "Banking systems, currency creation"},
	{RollMin: 97, RollMax: 97, SocialClass: "Trade Sovereignty", WealthKind: "personal", StartingCredit: 250000, Description: "Commercial control of regions"},
	{RollMin: 98, RollMax: 98, SocialClass: "Plutocrat Dynasty", WealthKind: "personal", StartingCredit: 275000, Description: "Wealth beyond noble titles"},
	{RollMin: 99, RollMax: 99, SocialClass: "Economic System Controller", WealthKind: "personal", StartingCredit: 300000, Description: "Market control, price-making power"},
	{RollMin: 100, RollMax: 100, SocialClass: "Merchant Paragon", WealthKind: "organizational", StartingCredit: ParentageOrganizationalCreditFloor, Description: "Legendary trade guild leadership, commands resources of thousands"},
	{RollMin: 101, RollMax: 120, SocialClass: "Organizational Wealth", WealthKind: "organizational", StartingCredit: ParentageOrganizationalCreditFloor, Description: "Various levels of institutional/corporate control beyond personal wealth"},
}

func ParentageChartEntryForRoll(roll int) (ParentageChartEntry, bool) {
	for _, entry := range ParentageChartV11 {
		if roll >= entry.RollMin && roll <= entry.RollMax {
			return entry, true
		}
	}
	return ParentageChartEntry{}, false
}

func ValidateParentageChartV11() error {
	if ParentageChartVersionV11 != "1.1" {
		return fmt.Errorf("unexpected parentage chart version %q", ParentageChartVersionV11)
	}
	if len(ParentageChartV11) == 0 {
		return fmt.Errorf("parentage chart is empty")
	}

	expectedRoll := 3
	for i, entry := range ParentageChartV11 {
		if entry.RollMin > entry.RollMax {
			return fmt.Errorf("parentage entry %d has inverted roll range %d-%d", i, entry.RollMin, entry.RollMax)
		}
		if strings.TrimSpace(entry.SocialClass) == "" {
			return fmt.Errorf("parentage entry %d is missing a social class", i)
		}
		if strings.TrimSpace(entry.WealthKind) == "" {
			return fmt.Errorf("parentage entry %d is missing a wealth kind", i)
		}
		if strings.TrimSpace(entry.Description) == "" {
			return fmt.Errorf("parentage entry %d is missing a description", i)
		}
		if entry.RollMin != expectedRoll {
			return fmt.Errorf("parentage entry %d starts at %d, expected %d", i, entry.RollMin, expectedRoll)
		}
		if entry.WealthKind != "personal" && entry.WealthKind != "organizational" {
			return fmt.Errorf("parentage entry %d has invalid wealth kind %q", i, entry.WealthKind)
		}
		if entry.WealthKind == "personal" && entry.RollMin >= 100 {
			return fmt.Errorf("personal parentage entry %d cannot be 100+", i)
		}
		if entry.WealthKind == "organizational" && entry.StartingCredit != ParentageOrganizationalCreditFloor {
			return fmt.Errorf("organizational parentage entry %d must use the organizational floor", i)
		}
		expectedRoll = entry.RollMax + 1
	}

	if expectedRoll != 121 {
		return fmt.Errorf("parentage chart ends at %d, expected 120", expectedRoll-1)
	}

	return nil
}
