from ml_service.features import DESIGN_FEATURES, fit_design_features


def test_fit_returns_expected_columns(design_base):
    model = fit_design_features(design_base)
    assert tuple(model.columns) == DESIGN_FEATURES


def test_transform_shapes_and_scales(design_base):
    model = fit_design_features(design_base)
    vec = model.transform(design_base[0])
    assert vec.shape == (len(DESIGN_FEATURES),)
    # Standardized features have mean ~0, std ~1 across the base.
    import numpy as np

    base = np.vstack([model.transform(r) for r in design_base])
    assert np.allclose(base.mean(axis=0), 0, atol=1e-9)
    assert np.allclose(base.std(axis=0), 1, atol=1e-9)


def test_transform_missing_key_defaults_zero(design_base):
    model = fit_design_features(design_base)
    vec = model.transform({"transformer_id": "TR-000"})
    assert vec.shape == (len(DESIGN_FEATURES),)
