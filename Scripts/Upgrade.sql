ALTER TABLE firefly.Electrolyser ADD COLUMN IF NOT EXISTS innerH2Pressure FLOAT NULL;
ALTER TABLE firefly.Electrolyser ADD COLUMN IF NOT EXISTS outerH2Pressure FLOAT NULL;
ALTER TABLE firefly.Electrolyser ADD COLUMN IF NOT EXISTS electrolyteLevel TINYINT NULL;
ALTER TABLE firefly.Electrolyser ADD COLUMN IF NOT EXISTS electrolyteFlow FLOAT NULL;
ALTER TABLE firefly.Electrolyser ADD COLUMN IF NOT EXISTS electronicFanSpeed FLOAT NULL;
ALTER TABLE firefly.Electrolyser ADD COLUMN IF NOT EXISTS airFanSpeed FLOAT NULL;
ALTER TABLE firefly.Electrolyser ADD COLUMN IF NOT EXISTS electrolyteFanSpeed FLOAT NULL;
ALTER TABLE firefly.Electrolyser ADD COLUMN IF NOT EXISTS downstreamTemperature FLOAT NULL;

ALTER TABLE firefly.Electrolyser_Archive ADD COLUMN IF NOT EXISTS innerH2Pressure FLOAT NULL;
ALTER TABLE firefly.Electrolyser_Archive ADD COLUMN IF NOT EXISTS outerH2Pressure FLOAT NULL;
ALTER TABLE firefly.Electrolyser_Archive ADD COLUMN IF NOT EXISTS electrolyteLevel TINYINT NULL;
ALTER TABLE firefly.Electrolyser_Archive ADD COLUMN IF NOT EXISTS electrolyteFlow FLOAT NULL;
ALTER TABLE firefly.Electrolyser_Archive ADD COLUMN IF NOT EXISTS electronicFanSpeed FLOAT NULL;
ALTER TABLE firefly.Electrolyser_Archive ADD COLUMN IF NOT EXISTS airFanSpeed FLOAT NULL;
ALTER TABLE firefly.Electrolyser_Archive ADD COLUMN IF NOT EXISTS electrolyteFanSpeed FLOAT NULL;
ALTER TABLE firefly.Electrolyser_Archive ADD COLUMN IF NOT EXISTS downstreamTemperature FLOAT NULL;
