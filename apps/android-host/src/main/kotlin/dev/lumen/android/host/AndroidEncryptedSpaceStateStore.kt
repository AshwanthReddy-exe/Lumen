package dev.lumen.android.host

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.AtomicFile
import dev.lumen.core.SpaceState
import dev.lumen.core.SpaceStateCodec
import dev.lumen.core.SpaceStateStore
import dev.lumen.core.SpaceStateStoreCommit
import dev.lumen.core.SpaceStateStoreInitialize
import dev.lumen.core.SpaceStateStoreRead
import dev.lumen.core.Transition
import java.io.File
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

class AndroidEncryptedSpaceStateStore(context: Context) : SpaceStateStore {
    private val file = AtomicFile(File(context.filesDir, "space-state.v1"))

    @Synchronized
    override fun read(): SpaceStateStoreRead = try {
        if (!file.baseFile.exists()) SpaceStateStoreRead.Missing else SpaceStateStoreRead.Present(
            SpaceStateCodec.decode(decrypt(file.openRead().use { it.readBytes() })),
        )
    } catch (_: Exception) {
        SpaceStateStoreRead.Unavailable
    }

    @Synchronized
    override fun initialize(state: SpaceState): SpaceStateStoreInitialize = when (read()) {
        is SpaceStateStoreRead.Present -> SpaceStateStoreInitialize.ALREADY_EXISTS
        SpaceStateStoreRead.Unavailable -> SpaceStateStoreInitialize.UNAVAILABLE
        SpaceStateStoreRead.Missing -> if (write(state)) SpaceStateStoreInitialize.CREATED else SpaceStateStoreInitialize.UNAVAILABLE
    }

    @Synchronized
    override fun transact(operation: (SpaceState) -> Transition): SpaceStateStoreCommit {
        val current = (read() as? SpaceStateStoreRead.Present)?.state ?: error("Host must open before transacting")
        val transition = operation(current)
        return if (write(transition.state)) SpaceStateStoreCommit.Committed(transition)
        else SpaceStateStoreCommit.Unavailable(current)
    }

    private fun write(state: SpaceState): Boolean = try {
        val stream = file.startWrite()
        try {
            stream.write(encrypt(SpaceStateCodec.encode(state)))
            file.finishWrite(stream)
            true
        } catch (failure: Exception) {
            file.failWrite(stream)
            false
        }
    } catch (_: Exception) {
        false
    }

    private fun encrypt(value: String): ByteArray {
        val cipher = Cipher.getInstance("AES/GCM/NoPadding")
        cipher.init(Cipher.ENCRYPT_MODE, key())
        return cipher.iv + cipher.doFinal(value.encodeToByteArray())
    }

    private fun decrypt(value: ByteArray): String {
        require(value.size > 12)
        val cipher = Cipher.getInstance("AES/GCM/NoPadding")
        cipher.init(Cipher.DECRYPT_MODE, key(), GCMParameterSpec(128, value.copyOfRange(0, 12)))
        return cipher.doFinal(value.copyOfRange(12, value.size)).decodeToString()
    }

    private fun key(): SecretKey {
        val store = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
        (store.getKey(KEY_ALIAS, null) as? SecretKey)?.let { return it }
        return KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore").apply {
            init(KeyGenParameterSpec.Builder(KEY_ALIAS, KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT)
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .build())
        }.generateKey()
    }

    private companion object { const val KEY_ALIAS = "lumen.space.state.v1" }
}
