package dev.lumen.core

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertIs

class SpaceHostTest {
    @Test
    fun `Host commits a command before returning it and restores honest state`() {
        val store = MemoryStore()
        val created = assertIs<SpaceHostOpenResult.Ready>(SpaceHost.create(store, "space", "owner", "host")).host
        val paired = assertIs<Transition.Applied>(created.pairNode(PairNodeCommand("pair", "owner", "worker")))
        assertEquals(paired.state, store.state)

        val advertised = assertIs<Transition.Applied>(created.advertiseCapability(
            AdvertiseCapabilityCommand("advertise", "worker", "worker", "coding.run", "apply"),
        ))
        assertEquals(advertised.state, store.state)
        val granted = assertIs<Transition.Applied>(created.setGrant(
            SetGrantCommand("allow", "owner", "worker", "coding.run", "apply", Grant.ALLOW),
        ))
        assertEquals(granted.state, store.state)
        assertIs<Transition.Applied>(created.submit(submit("queued")))

        val restarted = assertIs<SpaceHostOpenResult.Ready>(SpaceHost.open(store, "restart")).host
        val restored = assertIs<SpaceStateStoreRead.Present>(restarted.snapshot()).state
        assertEquals(TaskStatus.UNKNOWN_OUTCOME, restored.tasks.getValue("queued").status)
        assertEquals(restored, store.state)
    }

    @Test
    fun `Host keeps its last committed state when persistence fails`() {
        val store = MemoryStore()
        val host = assertIs<SpaceHostOpenResult.Ready>(SpaceHost.create(store, "space", "owner", "host")).host
        store.writable = false

        val rejected = assertIs<Transition.Rejected>(host.pairNode(PairNodeCommand("pair", "owner", "worker")))

        assertEquals(RejectionReason.PERSISTENCE_UNAVAILABLE, rejected.reason)
        store.writable = true
        assertEquals(setOf("owner", "host"), assertIs<SpaceStateStoreRead.Present>(host.snapshot()).state.nodes.keys)
        assertEquals(setOf("owner", "host"), store.state!!.nodes.keys)
    }

    @Test
    fun `Host stays unavailable when a recovered state cannot be committed`() {
        val store = MemoryStore()
        val host = assertIs<SpaceHostOpenResult.Ready>(SpaceHost.create(store, "space", "owner", "host")).host
        assertIs<Transition.Applied>(host.pairNode(PairNodeCommand("pair", "owner", "worker")))
        assertIs<Transition.Applied>(host.advertiseCapability(AdvertiseCapabilityCommand("advertise", "worker", "worker", "coding.run", "apply")))
        assertIs<Transition.Applied>(host.setGrant(SetGrantCommand("allow", "owner", "worker", "coding.run", "apply", Grant.ALLOW)))
        assertIs<Transition.Applied>(host.submit(submit("queued")))
        store.writable = false

        val opened = SpaceHost.open(store, "restart")

        assertEquals(SpaceHostOpenResult.Unavailable(HostUnavailableReason.PERSISTENCE_UNAVAILABLE), opened)
        assertEquals(TaskStatus.QUEUED, store.state!!.tasks.getValue("queued").status)
    }

    @Test
    fun `Host distinguishes a missing store from an unreadable one`() {
        assertEquals(
            SpaceHostOpenResult.Unavailable(HostUnavailableReason.STATE_NOT_FOUND),
            SpaceHost.open(MemoryStore(), "restart"),
        )
        val unreadable = MemoryStore().also { it.readable = false }
        assertEquals(
            SpaceHostOpenResult.Unavailable(HostUnavailableReason.PERSISTENCE_UNAVAILABLE),
            SpaceHost.open(unreadable, "restart"),
        )
    }

    @Test
    fun `Host creation cannot replace an existing Space`() {
        val store = MemoryStore()
        val first = assertIs<SpaceHostOpenResult.Ready>(SpaceHost.create(store, "space-one", "owner", "host"))

        val second = SpaceHost.create(store, "space-two", "owner-two", "host-two")

        assertEquals(SpaceHostOpenResult.Unavailable(HostUnavailableReason.STATE_ALREADY_EXISTS), second)
        assertEquals("space-one", assertIs<SpaceStateStoreRead.Present>(first.host.snapshot()).state.spaceId)
    }

    private fun submit(taskId: String) = SubmitCommand(
        "submit-$taskId", "space", 1, taskId, "owner", "worker", "coding.run", "apply", "digest",
    )

    private class MemoryStore : SpaceStateStore {
        var state: SpaceState? = null
        var writable = true
        var readable = true
        override fun read() = if (readable) state?.let(SpaceStateStoreRead::Present) ?: SpaceStateStoreRead.Missing else SpaceStateStoreRead.Unavailable
        override fun initialize(state: SpaceState): SpaceStateStoreInitialize {
            if (!writable) return SpaceStateStoreInitialize.UNAVAILABLE
            if (this.state != null) return SpaceStateStoreInitialize.ALREADY_EXISTS
            this.state = state
            return SpaceStateStoreInitialize.CREATED
        }
        override fun transact(operation: (SpaceState) -> Transition): SpaceStateStoreCommit {
            val current = state ?: error("missing state")
            if (!writable) return SpaceStateStoreCommit.Unavailable(current)
            val transition = operation(current)
            state = transition.state
            return SpaceStateStoreCommit.Committed(transition)
        }
    }
}
