-- Gold: dim_sensor — conformed sensor catalog (types, units, limits). Used by
-- reporting/BI to describe every measure. Static catalog, no rows from data.
{{ config(materialized='table') }}

select s.*
from (
    values
        ('load',                 'LOAD',         '%',      0,   200),
        ('ambient',              'TEMPERATURE',  '°C',   -20,    55),
        ('oil_temperature',      'TEMPERATURE',  '°C',   -20,   150),
        ('winding_temperature',  'TEMPERATURE',  '°C',   -20,   200),
        ('oil_level',            'LEVEL',        '%',      0,   100),
        ('current',              'CURRENT',      'A',      0,  null),
        ('voltage',              'VOLTAGE',      'kV',     0,  null)
) as s(sensor_name, sensor_type, unit, min_limit, max_limit)
