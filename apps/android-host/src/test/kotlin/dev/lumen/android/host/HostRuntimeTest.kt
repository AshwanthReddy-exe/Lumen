package dev.lumen.android.host

import org.junit.Assert.assertEquals
import org.junit.Test

class HostRuntimeTest {
    @Test
    fun `runtime exposes the latest explicit health state`() {
        HostRuntime.stopped()
        HostRuntime.opening()

        assertEquals(HostRuntimeStatus.OPENING, HostRuntime.state.value.status)

        HostRuntime.degraded("Encrypted state unavailable")

        assertEquals("Encrypted state unavailable", HostRuntime.state.value.detail)
    }
}
