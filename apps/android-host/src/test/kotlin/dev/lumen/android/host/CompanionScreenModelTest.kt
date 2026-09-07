package dev.lumen.android.host

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class CompanionScreenModelTest {
    @Test
    fun `an unconfigured phone directs its owner to create a Space`() {
        val screen = CompanionScreenModel.unconfigured()

        assertEquals("Lumen is ready to become your Host", screen.headline)
        assertEquals("Create your Space", screen.primaryAction)
        assertEquals("No microphone, camera, or network access is active.", screen.privacyNotice)
        assertTrue(screen.primaryActionEnabled)
    }

    @Test
    fun `a configured phone can start its Host service`() {
        val screen = CompanionScreenModel.readyToStart()

        assertEquals("Your Space is stored securely", screen.headline)
        assertEquals("Start Host", screen.primaryAction)
        assertTrue(screen.primaryActionEnabled)
    }
}
