package dev.lumen.android.host

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.sizeIn
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.testTagsAsResourceId
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

class LumenHostActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            LumenTheme {
                Surface(modifier = Modifier.fillMaxSize()) {
                    CompanionScreen(CompanionScreenModel.unconfigured())
                }
            }
        }
    }
}

@Composable
private fun CompanionScreen(screen: CompanionScreenModel) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .semantics { testTagsAsResourceId = true }
            .padding(PaddingValues(horizontal = 24.dp, vertical = 32.dp)),
        verticalArrangement = Arrangement.spacedBy(24.dp),
    ) {
        Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Text(
                text = "Desk companion",
                style = MaterialTheme.typography.labelLarge,
                color = MaterialTheme.colorScheme.primary,
            )
            Text(
                text = screen.headline,
                style = MaterialTheme.typography.headlineMedium,
                fontWeight = FontWeight.SemiBold,
            )
            Text(
                text = "This phone can show your Space wherever you use it. It is not a Host until its encrypted store is ready.",
                style = MaterialTheme.typography.bodyLarge,
            )
        }
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(
                modifier = Modifier.padding(20.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                Text("Privacy boundary", style = MaterialTheme.typography.titleMedium)
                Text(screen.privacyNotice, style = MaterialTheme.typography.bodyMedium)
            }
        }
        Button(
            onClick = {},
            enabled = false,
            modifier = Modifier
                .fillMaxWidth()
                .sizeIn(minHeight = 48.dp),
        ) {
            Text(screen.primaryAction)
        }
        Text(
            text = "Encrypted storage is the next setup step.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}
