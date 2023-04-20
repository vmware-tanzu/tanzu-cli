CREATE TABLE IF NOT EXISTS "AvailablePluginBinaries" (
		"PluginName"         TEXT NOT NULL,
		"Target"             TEXT NOT NULL,
		"Version"            TEXT NOT NULL,
		PRIMARY KEY("PluginName", "Target", "Version")
);

CREATE TABLE IF NOT EXISTS "AvailablePluginGroups" (
		"Vendor"             TEXT NOT NULL,
		"Publisher"          TEXT NOT NULL,
		"GroupName"          TEXT NOT NULL,
		PRIMARY KEY("Vendor", "Publisher", "GroupName")
);
 