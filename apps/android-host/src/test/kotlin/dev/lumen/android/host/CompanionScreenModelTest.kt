package dev.lumen.android.host

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Test

class CompanionScreenModelTest {
    @Test
    fun `the Android app remains a companion and cannot claim Host authority`() {
        val screen = CompanionScreenModel.quarantined()

        assertEquals("Companion setup is paused", screen.headline)
        assertEquals("Waiting for Host pairing", screen.primaryAction)
        assertEquals("No microphone, camera, or network access is active.", screen.privacyNotice)
        assertEquals(
            "This app cannot create or run the Space Host. Pairing returns after the headless Host is ready.",
            screen.supportingText,
        )
        assertFalse(screen.primaryActionEnabled)
    }
}
