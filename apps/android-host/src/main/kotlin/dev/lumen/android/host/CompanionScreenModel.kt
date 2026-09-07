package dev.lumen.android.host

data class CompanionScreenModel(
    val headline: String,
    val primaryAction: String,
    val privacyNotice: String,
) {
    companion object {
        fun unconfigured() = CompanionScreenModel(
            headline = "Lumen is ready to become your Host",
            primaryAction = "Create your Space",
            privacyNotice = "No microphone, camera, or network access is active.",
        )
    }
}
