<script setup>
import { onMounted, ref } from "vue";

const health = ref({ status: "checking" });
const items = ref([]);
const newItemName = ref("");
const createError = ref("");
const loading = ref(true);

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

  const res = await fetch("/api/items", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name: newItemName.value }),
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    createError.value = body.error || `request failed (${res.status})`;
    return;
  }

  newItemName.value = "";
  await fetchItems();
}

onMounted(async () => {
  await Promise.all([fetchHealth(), fetchItems()]);
  loading.value = false;
});
</script>

<template>
  <div class="min-h-screen bg-slate-50 py-10">
    <div class="mx-auto max-w-2xl space-y-6 px-4">
      <header class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-slate-900">
          Healthwatch
        </h1>
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
        class="flex gap-2"
        @submit.prevent="createItem"
      >
        <input
          v-model="newItemName"
          type="text"
          placeholder="New item name"
          class="flex-1 rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-slate-500 focus:outline-none"
        >
        <button
          type="submit"
          class="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-700"
        >
          Add
        </button>
      </form>
      <p
        v-if="createError"
        class="text-sm text-red-600"
      >
        {{ createError }}
      </p>

      <div class="rounded-md border border-slate-200 bg-white">
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
          No items yet.
        </p>
        <ul
          v-else
          class="divide-y divide-slate-100"
        >
          <li
            v-for="item in items"
            :key="item.id"
            class="flex items-center justify-between px-4 py-3"
          >
            <span class="text-sm text-slate-900">{{ item.name }}</span>
            <span class="text-xs text-slate-400">{{ new Date(item.created_at).toLocaleString() }}</span>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>
