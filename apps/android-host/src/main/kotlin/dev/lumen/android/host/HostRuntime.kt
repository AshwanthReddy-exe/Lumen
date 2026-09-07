package dev.lumen.android.host

import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

enum class HostRuntimeStatus { STOPPED, OPENING, READY, DEGRADED }

data class HostRuntimeState(
    val status: HostRuntimeStatus,
    val detail: String,
)

object HostRuntime {
    private val mutableState = MutableStateFlow(stoppedState())

    val state: StateFlow<HostRuntimeState> = mutableState.asStateFlow()

    fun opening() = publish(HostRuntimeState(HostRuntimeStatus.OPENING, "Opening encrypted Space state"))

    fun ready() = publish(HostRuntimeState(HostRuntimeStatus.READY, "Encrypted Space state recovered"))

    fun degraded(detail: String) = publish(HostRuntimeState(HostRuntimeStatus.DEGRADED, detail))

    fun stopped() = publish(stoppedState())

    fun publish(state: HostRuntimeState) {
        mutableState.value = state
    }

    private fun stoppedState() = HostRuntimeState(HostRuntimeStatus.STOPPED, "Host is stopped")
}
