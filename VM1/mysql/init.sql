CREATE TABLE actividades (
    actividad_id VARCHAR(64) NOT NULL,
    ubicacion VARCHAR(255) NOT NULL,
    peligrosidad VARCHAR(32) NOT NULL,
    integrantes JSON NOT NULL,
    fecha_emision DATETIME NOT NULL,
    PRIMARY KEY (actividad_id),
    CONSTRAINT chk_actividades_integrantes CHECK (
        JSON_TYPE(integrantes) = 'ARRAY'
        AND JSON_LENGTH(integrantes) >= 1
    )
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;