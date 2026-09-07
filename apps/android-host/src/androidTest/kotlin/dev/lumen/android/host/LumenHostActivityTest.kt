package dev.lumen.android.host

import androidx.compose.ui.test.assertIsNotEnabled
import androidx.compose.ui.test.assertIsDisplayed
import androidx.compose.ui.test.junit4.createAndroidComposeRule
import androidx.compose.ui.test.onNodeWithText
import org.junit.Rule
import org.junit.Test

class LumenHostActivityTest {
    @get:Rule
    val compose = createAndroidComposeRule<LumenHostActivity>()

    @Test
    fun firstRunShowsItsPrivacyBoundaryAndUnavailableSetupAction() {
        compose.onNodeWithText("Lumen is ready to become your Host").assertIsDisplayed()
        compose.onNodeWithText("No microphone, camera, or network access is active.").assertIsDisplayed()
        compose.onNodeWithText("Create your Space").assertIsDisplayed().assertIsNotEnabled()
    }
}
