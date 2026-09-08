package dev.lumen.android.host

data class CompanionScreenModel(
    val headline: String,
    val primaryAction: String,
    val primaryActionEnabled: Boolean,
    val privacyNotice: String,
    val supportingText: String,
) {
    companion object {
        fun quarantined() = CompanionScreenModel(
            headline = "Companion setup is paused",
            primaryAction = "Waiting for Host pairing",
            primaryActionEnabled = false,
            privacyNotice = "No microphone, camera, or network access is active.",
            supportingText = "This app cannot create or run the Space Host. Pairing returns after the headless Host is ready.",
        )
    }
}
