-- Gold: dim_time — time spine derived from silver measurement instants.
-- Attributes for time-based analytics (drill-downs, seasons, weekends).
{{ config(materialized='table') }}

with times as (
    select distinct ts as time_key
    from {{ ref('int_telemetry') }}
),

final as (
    select
        time_key,
        extract(second from time_key)::int as second,
        extract(minute from time_key)::int as minute,
        extract(hour from time_key)::int as hour,
        extract(day from time_key)::int as day,
        extract(month from time_key)::int as month,
        extract(year from time_key)::int as year,
        to_char(time_key, 'YYYY-MM-DD') as date,
        to_char(time_key, 'Day') as weekday,
        extract(dow from time_key)::int as day_of_week,  -- 0=Sunday
        case when extract(dow from time_key) in (0, 6) then true else false end as is_weekend
    from times
)

select * from final
