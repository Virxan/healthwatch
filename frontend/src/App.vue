<script setup>
import { computed, onMounted, onUnmounted, ref } from "vue";

const health = ref({ status: "checking" });
const items = ref([]);
const newItemName = ref("");
const newItemUrl = ref("");
const createError = ref("");
const creating = ref(false);
const loading = ref(true);
const search = ref("");
const statusFilter = ref("all");
const clearing = ref(false);
const showClearConfirm = ref(false);
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

async function clearAll() {
  clearing.value = true;
  try {
    await fetch("/api/items", { method: "DELETE" });
    showClearConfirm.value = false;
    await fetchItems();
  } finally {
    clearing.value = false;
  }
}

function itemStatus(item) {
  if (!item.last_checked_at) return "checking";
  return item.last_status === "up" ? "up" : "down";
}

const upCount = computed(
  () => items.value.filter((i) => itemStatus(i) === "up").length,
);

const downCount = computed(
  () => items.value.filter((i) => itemStatus(i) === "down").length,
);

const avgLatency = computed(() => {
  const vals = items.value
    .map((i) => i.last_latency_ms)
    .filter((v) => v != null);
  if (vals.length === 0) return null;
  return Math.round(vals.reduce((a, b) => a + b, 0) / vals.length);
});

const filteredItems = computed(() => {
  const q = search.value.trim().toLowerCase();
  return items.value.filter((item) => {
    if (statusFilter.value !== "all" && itemStatus(item) !== statusFilter.value) {
      return false;
    }
    if (!q) return true;
    return (
      item.name.toLowerCase().includes(q) ||
      item.url.toLowerCase().includes(q)
    );
  });
});

function latencyClass(item) {
  const ms = item.last_latency_ms;
  if (ms == null) return "text-slate-400";
  if (ms < 100) return "text-emerald-600";
  if (ms < 300) return "text-amber-600";
  return "text-rose-600";
}

function tlsClass(days) {
  if (days == null) return "text-slate-400";
  if (days < 14) return "text-rose-600";
  if (days < 30) return "text-amber-600";
  return "text-slate-500";
}

function formatLatency(item) {
  return item.last_latency_ms != null ? `${item.last_latency_ms} ms` : "—";
}

function formatTLS(item) {
  return item.tls_days_remaining != null ? `${item.tls_days_remaining} j` : "—";
}

function formatChecked(item) {
  if (!item.last_checked_at) return "—";
  const diff = Date.now() - new Date(item.last_checked_at).getTime();
  // Guard against minor clock skew between the server and this browser,
  // which would otherwise show a nonsensical "il y a -87s".
  const s = Math.max(0, Math.round(diff / 1000));
  if (s < 5) return "à l'instant";
  if (s < 60) return `il y a ${s}s`;
  const m = Math.round(s / 60);
  if (m < 60) return `il y a ${m} min`;
  const h = Math.round(m / 60);
  if (h < 24) return `il y a ${h} h`;
  return `il y a ${Math.round(h / 24)} j`;
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
  <div class="min-h-screen bg-slate-50 text-slate-900">
    <div class="mx-auto max-w-5xl space-y-6 px-4 py-10">
      <header class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="grid h-10 w-10 place-items-center rounded-xl bg-slate-900 text-white">
            <svg
              class="h-5 w-5"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M3 12h4l2 5 4-10 2 5h6" />
            </svg>
          </div>
          <div>
            <h1 class="text-xl font-semibold leading-tight">
              Healthwatch
            </h1>
            <p class="text-sm text-slate-500">
              {{ items.length }} site(s) · vérifié toutes les 30s
            </p>
          </div>
        </div>
        <span
          class="inline-flex items-center gap-2 rounded-full px-3 py-1 text-sm font-medium"
          :class="{
            'bg-emerald-100 text-emerald-700': health.status === 'ok',
            'bg-rose-100 text-rose-700': health.status === 'down',
            'bg-slate-200 text-slate-600': health.status === 'checking',
          }"
        >
          <span class="h-2 w-2 rounded-full bg-current" />
          {{ health.status === "ok" ? "API healthy" : health.status === "down" ? "API down" : "Connexion..." }}
        </span>
      </header>

      <div class="grid grid-cols-2 gap-3 md:grid-cols-4">
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <p class="text-xs text-slate-500">
            Total surveillé
          </p>
          <p class="mt-1 text-2xl font-semibold">
            {{ items.length }}
          </p>
        </div>
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <p class="text-xs text-slate-500">
            En ligne
          </p>
          <p class="mt-1 text-2xl font-semibold text-emerald-600">
            {{ upCount }}
          </p>
        </div>
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <p class="text-xs text-slate-500">
            Hors ligne
          </p>
          <p class="mt-1 text-2xl font-semibold text-rose-600">
            {{ downCount }}
          </p>
        </div>
        <div class="rounded-xl border border-slate-200 bg-white p-4">
          <p class="text-xs text-slate-500">
            Latence moyenne
          </p>
          <p class="mt-1 text-2xl font-semibold">
            {{ avgLatency != null ? avgLatency + " ms" : "—" }}
          </p>
        </div>
      </div>

      <form
        class="flex flex-wrap gap-2 rounded-xl border border-slate-200 bg-white p-4"
        @submit.prevent="createItem"
      >
        <input
          v-model="newItemName"
          type="text"
          placeholder="Label (ex. Mon blog)"
          class="min-w-0 flex-1 rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
        >
        <input
          v-model="newItemUrl"
          type="text"
          placeholder="https://example.com"
          class="min-w-0 flex-1 rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
        >
        <button
          type="submit"
          :disabled="creating"
          class="rounded-lg bg-slate-900 px-5 py-2 text-sm font-medium text-white transition hover:bg-slate-700 disabled:opacity-50"
        >
          {{ creating ? "Vérification..." : "Ajouter" }}
        </button>
      </form>
      <p
        v-if="createError"
        class="-mt-3 text-sm text-rose-600"
      >
        {{ createError }}
      </p>

      <div class="flex flex-wrap items-center gap-2">
        <div class="relative min-w-0 flex-1">
          <svg
            class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <circle
              cx="11"
              cy="11"
              r="7"
            />
            <path d="m21 21-4.3-4.3" />
          </svg>
          <input
            v-model="search"
            type="text"
            placeholder="Rechercher un site..."
            class="w-full rounded-lg border border-slate-300 bg-white py-2 pl-9 pr-3 text-sm outline-none focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
          >
        </div>
        <div class="flex rounded-lg border border-slate-200 bg-white p-1">
          <button
            type="button"
            class="rounded-md px-3 py-1 text-sm transition"
            :class="statusFilter === 'all' ? 'bg-slate-900 text-white' : 'text-slate-600 hover:bg-slate-100'"
            @click="statusFilter = 'all'"
          >
            Tous
          </button>
          <button
            type="button"
            class="rounded-md px-3 py-1 text-sm transition"
            :class="statusFilter === 'up' ? 'bg-slate-900 text-white' : 'text-slate-600 hover:bg-slate-100'"
            @click="statusFilter = 'up'"
          >
            En ligne
          </button>
          <button
            type="button"
            class="rounded-md px-3 py-1 text-sm transition"
            :class="statusFilter === 'down' ? 'bg-slate-900 text-white' : 'text-slate-600 hover:bg-slate-100'"
            @click="statusFilter = 'down'"
          >
            Hors ligne
          </button>
        </div>
        <button
          type="button"
          :disabled="items.length === 0"
          class="rounded-lg border border-rose-200 px-3 py-2 text-sm font-medium text-rose-600 transition hover:bg-rose-50 disabled:opacity-40"
          @click="showClearConfirm = true"
        >
          Vider la base
        </button>
      </div>

      <div class="overflow-hidden rounded-xl border border-slate-200 bg-white">
        <p
          v-if="loading"
          class="p-6 text-center text-sm text-slate-500"
        >
          Chargement...
        </p>
        <p
          v-else-if="filteredItems.length === 0"
          class="p-6 text-center text-sm text-slate-500"
        >
          {{ items.length === 0 ? "Aucun site surveillé pour l'instant." : "Aucun résultat." }}
        </p>
        <table
          v-else
          class="w-full text-sm"
        >
          <thead>
            <tr class="border-b border-slate-100 bg-slate-50 text-left text-xs uppercase tracking-wide text-slate-400">
              <th class="px-4 py-3 font-medium">
                Site
              </th>
              <th class="px-4 py-3 font-medium">
                Statut
              </th>
              <th class="px-4 py-3 font-medium">
                Latence
              </th>
              <th class="px-4 py-3 font-medium">
                TLS
              </th>
              <th class="px-4 py-3 font-medium">
                Dernier check
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr
              v-for="item in filteredItems"
              :key="item.id"
              class="transition-colors hover:bg-slate-50"
            >
              <td class="px-4 py-3">
                <div class="font-medium text-slate-900">
                  {{ item.name }}
                </div>
                <a
                  :href="item.url"
                  target="_blank"
                  rel="noopener"
                  class="text-xs text-slate-400 hover:underline"
                >
                  {{ item.url }}
                </a>
              </td>
              <td class="px-4 py-3">
                <span
                  class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium"
                  :class="{
                    'bg-emerald-100 text-emerald-700': itemStatus(item) === 'up',
                    'bg-rose-100 text-rose-700': itemStatus(item) === 'down',
                    'bg-slate-100 text-slate-500': itemStatus(item) === 'checking',
                  }"
                >
                  <span class="h-1.5 w-1.5 rounded-full bg-current" />
                  {{ itemStatus(item) === "up" ? "up" : itemStatus(item) === "down" ? "down" : "..." }}
                </span>
              </td>
              <td
                class="px-4 py-3 font-medium"
                :class="latencyClass(item)"
              >
                {{ formatLatency(item) }}
              </td>
              <td
                class="px-4 py-3"
                :class="tlsClass(item.tls_days_remaining)"
              >
                {{ formatTLS(item) }}
              </td>
              <td class="px-4 py-3 text-xs text-slate-400">
                {{ formatChecked(item) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div
      v-if="showClearConfirm"
      class="fixed inset-0 z-10 flex items-center justify-center bg-slate-900/40 px-4"
    >
      <div class="w-full max-w-sm rounded-xl border border-slate-200 bg-white p-5 shadow-xl">
        <h2 class="text-lg font-semibold text-slate-900">
          Vider la base ?
        </h2>
        <p class="mt-2 text-sm text-slate-500">
          Cette action supprime définitivement les {{ items.length }} site(s)
          surveillé(s). Elle est irréversible.
        </p>
        <div class="mt-5 flex justify-end gap-2">
          <button
            type="button"
            class="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-600 transition hover:bg-slate-50"
            @click="showClearConfirm = false"
          >
            Annuler
          </button>
          <button
            type="button"
            :disabled="clearing"
            class="rounded-lg bg-rose-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-rose-700 disabled:opacity-50"
            @click="clearAll"
          >
            {{ clearing ? "Suppression..." : "Vider la base" }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
