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
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.semantics.testTagsAsResourceId
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import dev.lumen.core.SpaceHost
import dev.lumen.core.SpaceHostOpenResult
import dev.lumen.core.SpaceStateStoreRead
import java.util.UUID

class LumenHostActivity : ComponentActivity() {
    private val store by lazy { AndroidEncryptedSpaceStateStore(this) }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            val runtime by HostRuntime.state.collectAsState()
            val screen = CompanionScreenModel.from(storageState(), runtime)
            LumenTheme {
                Surface(modifier = Modifier.fillMaxSize()) {
                    CompanionScreen(screen) {
                        if (runtime.status == HostRuntimeStatus.READY) LumenHostService.stop(this)
                        else startOrCreateHost()
                    }
                }
            }
        }
    }

    private fun storageState() = when (store.read()) {
        SpaceStateStoreRead.Missing -> CompanionStorageState.MISSING
        is SpaceStateStoreRead.Present -> CompanionStorageState.PRESENT
        SpaceStateStoreRead.Unavailable -> CompanionStorageState.UNAVAILABLE
    }

    private fun startOrCreateHost() {
        when (store.read()) {
            SpaceStateStoreRead.Missing -> createSpace()
            is SpaceStateStoreRead.Present -> startHost()
            SpaceStateStoreRead.Unavailable -> HostRuntime.degraded("Encrypted Space state is unavailable")
        }
    }

    private fun createSpace() {
        val identity = UUID.randomUUID().toString()
        when (SpaceHost.create(store, "space-$identity", "owner-$identity", "host-$identity")) {
            is SpaceHostOpenResult.Ready -> startHost()
            is SpaceHostOpenResult.Unavailable -> if (store.read() is SpaceStateStoreRead.Present) startHost()
            else HostRuntime.degraded("Encrypted Space state is unavailable")
        }
    }

    private fun startHost() {
        HostRuntime.opening()
        LumenHostService.start(this)
    }
}

@Composable
private fun CompanionScreen(screen: CompanionScreenModel, onPrimaryAction: () -> Unit) {
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
                text = screen.supportingText,
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
            onClick = onPrimaryAction,
            enabled = screen.primaryActionEnabled,
            modifier = Modifier
                .fillMaxWidth()
                .sizeIn(minHeight = 48.dp),
        ) {
            Text(screen.primaryAction)
        }
        Text(
            text = "A companion can run on any paired node; this phone is only the first Host deployment.",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}
