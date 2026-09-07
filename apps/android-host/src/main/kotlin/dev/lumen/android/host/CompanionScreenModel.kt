package dev.lumen.android.host

enum class CompanionStorageState { MISSING, PRESENT, UNAVAILABLE }

data class CompanionScreenModel(
    val headline: String,
    val primaryAction: String,
    val primaryActionEnabled: Boolean,
    val privacyNotice: String,
    val supportingText: String,
) {
    companion object {
        fun from(
            storage: CompanionStorageState,
            runtime: HostRuntimeState,
        ): CompanionScreenModel = when (storage) {
            CompanionStorageState.MISSING -> unconfigured()
            CompanionStorageState.UNAVAILABLE -> unavailable()
            CompanionStorageState.PRESENT -> when (runtime.status) {
                HostRuntimeStatus.STOPPED -> readyToStart()
                HostRuntimeStatus.OPENING -> starting()
                HostRuntimeStatus.READY -> active(runtime.detail)
                HostRuntimeStatus.DEGRADED -> unavailable(runtime.detail)
            }
        }

        fun unconfigured() = CompanionScreenModel(
            headline = "Lumen is ready to become your Host",
            primaryAction = "Create your Space",
            primaryActionEnabled = true,
            privacyNotice = "No microphone, camera, or network access is active.",
            supportingText = "Creating a Space writes its encrypted state only on this device.",
        )

        fun readyToStart() = CompanionScreenModel(
            headline = "Your Space is stored securely",
            primaryAction = "Start Host",
            primaryActionEnabled = true,
            privacyNotice = "No microphone, camera, or network access is active.",
            supportingText = "Starting the Host keeps Space coordination active with a visible Android notification.",
        )

        fun starting() = CompanionScreenModel(
            headline = "Starting your Host",
            primaryAction = "Host is starting",
            primaryActionEnabled = false,
            privacyNotice = "No microphone, camera, or network access is active.",
            supportingText = "Lumen is opening the encrypted Space state before accepting future node work.",
        )

        fun active(detail: String) = CompanionScreenModel(
            headline = "Host is ready",
            primaryAction = "Stop Host",
            primaryActionEnabled = true,
            privacyNotice = "No microphone, camera, or network access is active.",
            supportingText = detail,
        )

        fun unavailable(detail: String = "Lumen did not start. Your encrypted state has not been replaced or reset.") = CompanionScreenModel(
            headline = "Host storage is unavailable",
            primaryAction = "Try again",
            primaryActionEnabled = true,
            privacyNotice = "No microphone, camera, or network access is active.",
            supportingText = detail,
        )
    }
}
