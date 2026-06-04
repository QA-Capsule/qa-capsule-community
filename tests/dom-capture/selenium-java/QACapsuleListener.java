package io.qacapsule.listener;

import org.openqa.selenium.OutputType;
import org.openqa.selenium.TakesScreenshot;
import org.openqa.selenium.WebDriver;
import org.junit.jupiter.api.extension.*;

import java.lang.reflect.Field;

/**
 * QA Capsule — Selenium Java DOM Capture (JUnit 5 Extension)
 * Captures page HTML + screenshot on every test failure.
 *
 * Usage — add to your test class:
 *   @ExtendWith(QACapsuleListener.class)
 *   class LoginTest {
 *       WebDriver driver;
 *       // tests...
 *   }
 *
 * Or register globally in src/test/resources/junit-platform.properties:
 *   junit.jupiter.extensions.autodetection.enabled=true
 */
public class QACapsuleListener implements TestWatcher, AfterEachCallback {

    private static final int MAX_HTML = 24_000;

    @Override
    public void testFailed(ExtensionContext context, Throwable cause) {
        WebDriver driver = resolveDriver(context);
        if (driver == null) return;
        captureDOM(driver);
    }

    @Override
    public void afterEach(ExtensionContext context) {
        // afterEach is a safety net — testFailed handles the DOM capture
    }

    // ── internal ──────────────────────────────────────────────────────────────

    private void captureDOM(WebDriver driver) {
        try {
            String html = driver.getPageSource();
            String trimmed = html.length() > MAX_HTML ? html.substring(0, MAX_HTML) : html;
            System.out.println("\n[QA_CAPSULE_DOM_SNAPSHOT_START]");
            System.out.println(trimmed);
            System.out.println("[QA_CAPSULE_DOM_SNAPSHOT_END]");
        } catch (Exception e) {
            System.err.println("[QA Capsule] DOM capture failed: " + e.getMessage());
        }

        try {
            if (driver instanceof TakesScreenshot) {
                byte[] screenshot = ((TakesScreenshot) driver).getScreenshotAs(OutputType.BYTES);
                System.out.println("[QA_CAPSULE_SCREENSHOT]: captured in memory (" + screenshot.length + " bytes)");
            }
        } catch (Exception ignored) {}
    }

    private WebDriver resolveDriver(ExtensionContext context) {
        return context.getTestInstance()
            .map(instance -> {
                for (Field field : instance.getClass().getDeclaredFields()) {
                    if (WebDriver.class.isAssignableFrom(field.getType())) {
                        field.setAccessible(true);
                        try { return (WebDriver) field.get(instance); } catch (Exception ignored) {}
                    }
                }
                return null;
            })
            .orElse(null);
    }
}
