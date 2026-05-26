import pytest


@pytest.mark.selenium
def test_driver_placeholder():
    # Placeholder keeps sample lightweight while producing JUnit shape.
    assert True


@pytest.mark.selenium
def test_selenium_intentional_failure():
    assert "chrome" == "firefox"
