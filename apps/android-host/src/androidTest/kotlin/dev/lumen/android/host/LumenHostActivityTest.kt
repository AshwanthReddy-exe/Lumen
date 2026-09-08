package dev.lumen.android.host

import android.Manifest
import android.content.pm.PackageManager
import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithText
import org.junit.Rule
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class LumenHostActivityTest {
    @get:Rule
    val compose = createAndroidComposeRule<LumenHostActivity>()

    @Test
    fun firstRunShowsItsPrivacyBoundaryAndUnavailableSetupAction() {
        compose.onNodeWithText("Companion setup is paused").assertIsDisplayed()
        compose.onNodeWithText("No microphone, camera, or network access is active.").assertIsDisplayed()
        compose.onNodeWithText("Waiting for Host pairing").assertIsDisplayed().assertIsNotEnabled()
    }

    @Test
    fun installedCompanionHasNoHostAuthoritySurface() {
        val packageInfo = compose.activity.packageManager.getPackageInfo(
            compose.activity.packageName,
            PackageManager.PackageInfoFlags.of(
                (PackageManager.GET_PERMISSIONS or PackageManager.GET_SERVICES).toLong(),
            ),
        )

        assertFalse(
            packageInfo.requestedPermissions.orEmpty().contains(Manifest.permission.FOREGROUND_SERVICE),
        )
        assertTrue(packageInfo.services.isNullOrEmpty())
        assertTrue(runCatching { Class.forName("dev.lumen.core.SpaceHost") }.isFailure)
    }
}
