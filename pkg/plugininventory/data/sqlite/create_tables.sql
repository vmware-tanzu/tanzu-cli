CREATE TABLE IF NOT EXISTS "PluginBinaries" (
		"PluginName"         TEXT NOT NULL,
		"Target"             TEXT NOT NULL,
		"RecommendedVersion" TEXT NOT NULL,
		"Version"            TEXT NOT NULL,
		"Hidden"             TEXT NOT NULL,
		"Description"        TEXT NOT NULL,
		"Publisher"          TEXT NOT NULL,
		"Vendor"             TEXT NOT NULL,
		"OS"                 TEXT NOT NULL,
		"Architecture"       TEXT NOT NULL,
		"Digest"             TEXT NOT NULL,
		"URI"                TEXT NOT NULL,
		PRIMARY KEY("PluginName", "Target", "Version", "OS", "Architecture")
);

CREATE TABLE IF NOT EXISTS "PluginGroups" (
		"Vendor"             TEXT NOT NULL,
		"Publisher"          TEXT NOT NULL,
		"GroupName"          TEXT NOT NULL,
		"PluginName"         TEXT NOT NULL,
		"Target"             TEXT NOT NULL,
		"Version"            TEXT NOT NULL,
		"Mandatory"          TEXT NOT NULL,
		"Hidden"             TEXT NOT NULL,
		PRIMARY KEY("Vendor", "Publisher", "GroupName", "PluginName", "Target", "Version")
);
