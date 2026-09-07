package dev.lumen.core

import kotlin.test.Test
import kotlin.test.assertEquals

class SpaceStateCodecTest {
    @Test
    fun `codec round trips canonical Space state`() {
        val state = Space.createSpace("space", "owner", "host")
        assertEquals(state, SpaceStateCodec.decode(SpaceStateCodec.encode(state)))
    }
}
