def test_api_ok():
    assert 200 == 200


def test_intentional_failure():
    assert 400 == 201
