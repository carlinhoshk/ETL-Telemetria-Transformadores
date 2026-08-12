from ml_service.features import fit_design_features
from ml_service.similarity import similarity_scores


def test_self_is_most_similar(design_base):
    target = design_base[0]
    candidates = [r for r in design_base if r["transformer_id"] != target["transformer_id"]]
    model = fit_design_features(design_base)
    scores = similarity_scores(target, candidates, model)
    assert len(scores) == len(candidates)
    # A duplicate of the target (same vector) must score first at ~1.0.
    scores_with_self = similarity_scores(target, [target, *candidates], model)
    top_id, top_score = scores_with_self[0]
    assert top_id == target["transformer_id"]
    assert top_score >= 0.99


def test_scores_are_bounded_and_sorted(design_base):
    target = design_base[2]
    candidates = [r for r in design_base if r["transformer_id"] != target["transformer_id"]]
    model = fit_design_features(design_base)
    scores = similarity_scores(target, candidates, model)
    values = [s for _, s in scores]
    assert all(0.0 < v <= 1.0 for v in values)
    assert values == sorted(values, reverse=True)
    # ids are all distinct and present.
    ids = [i for i, _ in scores]
    assert len(set(ids)) == len(ids)
