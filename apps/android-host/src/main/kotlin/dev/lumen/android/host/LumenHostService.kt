package dev.lumen.android.host

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.IBinder
import androidx.core.app.NotificationCompat
import dev.lumen.core.SpaceHost
import dev.lumen.core.SpaceHostOpenResult

class LumenHostService : Service() {
    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        startForeground(NOTIFICATION_ID, notification())
        val store = AndroidEncryptedSpaceStateStore(this)
        when (SpaceHost.open(store, "android-startup")) {
            is SpaceHostOpenResult.Ready -> {
                getSystemService(NotificationManager::class.java).notify(NOTIFICATION_ID, activeNotification())
            }
            is SpaceHostOpenResult.Unavailable -> stopSelf(startId)
        }
        return START_NOT_STICKY
    }

    override fun onBind(intent: Intent?): IBinder? = null

    private fun notification() = notification("Opening encrypted Space state")

    private fun activeNotification() = notification("Secure Space coordination is active")

    private fun notification(text: String) = NotificationCompat.Builder(this, CHANNEL_ID)
        .setSmallIcon(android.R.drawable.stat_notify_sync)
        .setContentTitle("Lumen Host")
        .setContentText(text)
        .setOngoing(true)
        .build()

    override fun onCreate() {
        super.onCreate()
        getSystemService(NotificationManager::class.java).createNotificationChannel(
            NotificationChannel(CHANNEL_ID, "Lumen Host", NotificationManager.IMPORTANCE_LOW),
        )
    }

    companion object {
        private const val CHANNEL_ID = "lumen-host"
        private const val NOTIFICATION_ID = 1001
        fun start(context: Context) = context.startForegroundService(Intent(context, LumenHostService::class.java))
    }
}
