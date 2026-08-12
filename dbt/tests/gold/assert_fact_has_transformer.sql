-- Gold data test: every measurement row must resolve to a transformer in the
-- conformed dimension.
select f.transformer_key
from {{ ref('fact_transformer_measurement') }} f
left join {{ ref('dim_transformer') }} d on d.transformer_key = f.transformer_key
where d.transformer_key is null
