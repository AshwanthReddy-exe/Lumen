package dev.lumen.core

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

object SpaceStateCodec {
    private val json = Json {
        ignoreUnknownKeys = false
        encodeDefaults = true
        allowStructuredMapKeys = true
    }

    fun encode(state: SpaceState): String = json.encodeToString(SpaceState.serializer(), state)
    fun decode(value: String): SpaceState = json.decodeFromString(SpaceState.serializer(), value)
}
