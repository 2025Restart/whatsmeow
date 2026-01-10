-- v12: Device fingerprint customization
-- 设备指纹信息表
CREATE TABLE IF NOT EXISTS whatsmeow_device_fingerprint (
    jid TEXT PRIMARY KEY,
    
    -- UserAgent 字段
    manufacturer TEXT,
    device TEXT,
    device_model_type TEXT,
    os_version TEXT,
    os_build_number TEXT,
    mcc TEXT,
    mnc TEXT,
    locale_language TEXT,
    locale_country TEXT,
    platform INTEGER,  -- waWa6.ClientPayload_UserAgent_Platform 枚举值
    app_version_primary INTEGER,
    app_version_secondary INTEGER,
    app_version_tertiary INTEGER,
    device_type INTEGER,  -- waWa6.ClientPayload_UserAgent_DeviceType 枚举值
    device_board TEXT,
    
    -- DeviceProps 字段
    device_props_os TEXT,
    device_props_version_primary INTEGER,
    device_props_version_secondary INTEGER,
    device_props_version_tertiary INTEGER,
    platform_type INTEGER,  -- waCompanionReg.DeviceProps_PlatformType 枚举值
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_device_fingerprint_jid ON whatsmeow_device_fingerprint(jid);
