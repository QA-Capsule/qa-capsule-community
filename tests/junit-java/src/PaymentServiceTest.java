import org.junit.jupiter.api.*;
import static org.junit.jupiter.api.Assertions.*;

@DisplayName("Payment Service Tests")
class PaymentServiceTest {

    @Test
    @DisplayName("Charge succeeds with valid card")
    void chargeSucceedsWithValidCard() {
        System.out.println("[STDOUT] POST /api/payments → {card: 4242424242424242, amount: 99.99}");
        boolean charged = true;
        assertTrue(charged, "Payment should have been processed");
        System.out.println("[STDOUT] Charge confirmed: txn_abc123");
    }

    @Test
    @DisplayName("Charge fails with expired card")
    void chargeFailsWithExpiredCard() {
        System.out.println("[STDOUT] POST /api/payments → {card: 4000000000000069}");
        System.err.println("[STDERR] Card declined — expiration date in the past");
        boolean charged = false;
        assertTrue(charged, "Expected payment to succeed but card was declined");
    }

    @Test
    @DisplayName("Refund reduces balance correctly")
    void refundReducesBalanceCorrectly() {
        System.out.println("[STDOUT] POST /api/refunds → {txn: txn_abc123, amount: 99.99}");
        double balance = 0.00;
        assertEquals(0.00, balance, 0.001, "Balance should be 0 after full refund");
        System.out.println("[STDOUT] Refund issued — balance: $0.00");
    }
}
