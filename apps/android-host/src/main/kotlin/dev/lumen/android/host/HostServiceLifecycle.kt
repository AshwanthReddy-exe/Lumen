package dev.lumen.android.host

import dev.lumen.core.SpaceHostOpenResult

object HostServiceLifecycle {
    fun afterOpen(result: SpaceHostOpenResult): HostRuntimeState = when (result) {
        is SpaceHostOpenResult.Ready -> HostRuntimeState(
            HostRuntimeStatus.READY,
            "Encrypted Space state recovered",
        )
        is SpaceHostOpenResult.Unavailable -> HostRuntimeState(
            HostRuntimeStatus.DEGRADED,
            "Encrypted Space state is unavailable",
        )
    }

    fun afterStop() = HostRuntimeState(HostRuntimeStatus.STOPPED, "Host is stopped")

    fun afterTimeout() = HostRuntimeState(
        HostRuntimeStatus.DEGRADED,
        "Android stopped continuous coordination; restart required",
    )
}
