package dev.lumen.android.host

import dev.lumen.core.HostUnavailableReason
import dev.lumen.core.SpaceHostOpenResult
import org.junit.Assert.assertEquals
import org.junit.Test

class HostServiceLifecycleTest {
    @Test
    fun `an unavailable open becomes degraded rather than ready`() {
        val state = HostServiceLifecycle.afterOpen(
            SpaceHostOpenResult.Unavailable(HostUnavailableReason.PERSISTENCE_UNAVAILABLE),
        )

        assertEquals(HostRuntimeStatus.DEGRADED, state.status)
        assertEquals("Encrypted Space state is unavailable", state.detail)
    }
}
