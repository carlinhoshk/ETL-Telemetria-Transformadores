-- Custom generic test: a model's column set must be unique.
{% test unique_telemetry_key(model, columns) %}
    select *
    from (
        select {{ columns | join(', ') }}, count(*) as cnt
        from {{ model }}
        group by {{ columns | join(', ') }}
        having count(*) > 1
    ) d
{% endtest %}
