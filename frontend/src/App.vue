<script setup>
import { onMounted, onUnmounted, ref } from "vue";

const health = ref({ status: "checking" });
const items = ref([]);
const newItemName = ref("");
const newItemUrl = ref("");
const createError = ref("");
const creating = ref(false);
const loading = ref(true);
let refreshTimer = null;

async function fetchHealth() {
  try {
    const res = await fetch("/api/health");
    health.value = await res.json();
  } catch {
    health.value = { status: "down", error: "request failed" };
  }
}

async function fetchItems() {
  const res = await fetch("/api/items");
  items.value = await res.json();
}

async function createItem() {
  createError.value = "";
  creating.value = true;

  try {
    const res = await fetch("/api/items", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: newItemName.value, url: newItemUrl.value }),
    });

    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      createError.value = body.error || `request failed (${res.status})`;
      return;
    }

    newItemName.value = "";
    newItemUrl.value = "";
    await fetchItems();
  } finally {
    creating.value = false;
  }
}

function statusLabel(item) {
  if (!item.last_checked_at) return "checking...";
  return item.last_status === "up" ? "up" : "down";
}

function formatLatency(item) {
  return item.last_latency_ms != null ? `${item.last_latency_ms} ms` : "-";
}

function formatTLS(item) {
  return item.tls_days_remaining != null ? `${item.tls_days_remaining} days` : "-";
}

onMounted(async () => {
  await Promise.all([fetchHealth(), fetchItems()]);
  loading.value = false;
  // The backend re-checks every 30s; poll a bit more often so newly
  // checked statuses show up without a manual refresh.
  refreshTimer = setInterval(() => {
    fetchHealth();
    fetchItems();
  }, 10000);
});

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer);
});
</script>

<template>
  <div class="min-h-screen bg-slate-50 py-10">
    <div class="mx-auto max-w-3xl space-y-6 px-4">
      <header class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-slate-900">
            Healthwatch
          </h1>
          <p class="text-sm text-slate-500">
            Watching {{ items.length }} site(s)
          </p>
        </div>
        <span
          class="rounded-full px-3 py-1 text-sm font-medium"
          :class="{
            'bg-green-100 text-green-800': health.status === 'ok',
            'bg-red-100 text-red-800': health.status === 'down',
            'bg-slate-200 text-slate-600': health.status === 'checking',
          }"
        >
          {{ health.status === "ok" ? "Healthy" : health.status === "down" ? "Unhealthy" : "Checking..." }}
        </span>
      </header>

      <form
        class="flex flex-wrap gap-2"
        @submit.prevent="createItem"
      >
        <input
          v-model="newItemName"
          type="text"
          placeholder="Label (e.g. My blog)"
          class="min-w-0 flex-1 rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none"
        >
        <input
          v-model="newItemUrl"
          type="text"
          placeholder="https://example.com"
          class="min-w-0 flex-1 rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none"
        >
        <button
          type="submit"
          :disabled="creating"
          class="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700 disabled:opacity-50"
        >
          {{ creating ? "Checking..." : "Add" }}
        </button>
      </form>
      <p
        v-if="createError"
        class="text-sm text-red-600"
      >
        {{ createError }}
      </p>

      <div class="overflow-x-auto rounded-md border border-slate-200 bg-white">
        <p
          v-if="loading"
          class="p-4 text-sm text-slate-500"
        >
          Loading...
        </p>
        <p
          v-else-if="items.length === 0"
          class="p-4 text-sm text-slate-500"
        >
          No sites watched yet.
        </p>
        <table
          v-else
          class="w-full text-sm"
        >
          <thead>
            <tr class="border-b border-slate-100 text-left text-slate-500">
              <th class="px-4 py-2 font-medium">
                Site
              </th>
              <th class="px-4 py-2 font-medium">
                Status
              </th>
              <th class="px-4 py-2 font-medium">
                Latency
              </th>
              <th class="px-4 py-2 font-medium">
                TLS expiry
              </th>
              <th class="px-4 py-2 font-medium">
                Last checked
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr
              v-for="item in items"
              :key="item.id"
            >
              <td class="px-4 py-3">
                <div class="text-slate-900">
                  {{ item.name }}
                </div>
                <a
                  :href="item.url"
                  target="_blank"
                  rel="noopener"
                  class="text-xs text-slate-400 hover:underline"
                >{{
                  item.url
                }}</a>
              </td>
              <td class="px-4 py-3">
                <span
                  class="font-medium"
                  :class="{
                    'text-green-700': statusLabel(item) === 'up',
                    'text-red-700': statusLabel(item) === 'down',
                    'text-slate-400': statusLabel(item) === 'checking...',
                  }"
                >
                  {{ statusLabel(item) }}
                </span>
              </td>
              <td class="px-4 py-3 text-slate-600">
                {{ formatLatency(item) }}
              </td>
              <td class="px-4 py-3 text-slate-600">
                {{ formatTLS(item) }}
              </td>
              <td class="px-4 py-3 text-xs text-slate-400">
                {{ item.last_checked_at ? new Date(item.last_checked_at).toLocaleString() : "-" }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
