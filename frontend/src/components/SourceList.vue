<script setup>
import { ref, onMounted } from 'vue';

const sources = ref([]);
const newSourceUrl = ref('');
const selectedSourceId = ref(null);

const emit = defineEmits(['source-selected']);

// Fetch all sources when the component is mounted
onMounted(async () => {
  try {
    const response = await fetch('/api/sources');
    if (!response.ok) {
      throw new Error('Network response was not ok');
    }
    sources.value = await response.json();
  } catch (error) {
    console.error('Failed to fetch sources:', error);
  }
});

// Add a new source
const addSource = async () => {
  if (!newSourceUrl.value.trim()) {
    return;
  }
  try {
    const response = await fetch('/api/sources', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ url: newSourceUrl.value }),
    });

    if (response.status === 409) {
      alert('Feed source already exists.');
      return;
    }

    if (!response.ok) {
      throw new Error('Failed to add source');
    }

    const newSource = await response.json();
    sources.value.push(newSource);
    newSourceUrl.value = ''; // Clear input
  } catch (error) {
    console.error('Error adding source:', error);
    alert('Failed to add feed source. Check the URL and try again.');
  }
};

const selectSource = (source) => {
  selectedSourceId.value = source.id;
  emit('source-selected', source);
};

// Implement deleteSource function
const deleteSource = async (sourceId, event) => {
  event.stopPropagation(); // Prevent li click event from firing
  if (!confirm('Are you sure you want to delete this feed and all its articles?')) {
    return;
  }
  try {
    const response = await fetch(`/api/sources/${sourceId}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      throw new Error('Failed to delete source');
    }
    sources.value = sources.value.filter((s) => s.id !== sourceId);
    // If the deleted source was the selected one, clear the selection
    if (selectedSourceId.value === sourceId) {
      selectedSourceId.value = null;
      emit('source-selected', null);
    }
  } catch (error) {
    console.error('Error deleting source:', error);
    alert('Failed to delete feed source.');
  }
};
</script>

<template>
  <div class="source-list-container">
    <h2>Feeds</h2>
    <div class="add-source-form">
      <input
        type="text"
        v-model="newSourceUrl"
        placeholder="Enter RSS feed URL"
        @keyup.enter="addSource"
      />
      <button @click="addSource">Add</button>
    </div>
    <ul>
      <li
        v-for="source in sources"
        :key="source.id"
        @click="selectSource(source)"
        :class="{ selected: source.id === selectedSourceId }"
      >
        <span>{{ source.name }}</span>
        <button class="delete-btn" @click="deleteSource(source.id, $event)">×</button>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.source-list-container {
  padding: 1rem;
  border-right: 1px solid var(--color-border);
  height: 100vh;
  overflow-y: auto;
  background-color: var(--color-bg-pane);
}

h2 {
  margin-top: 0;
  font-size: 1.2rem;
  color: var(--color-text-primary);
}

.add-source-form {
  display: flex;
  margin-bottom: 1rem;
}

input[type="text"] {
  flex-grow: 1;
  padding: 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: 4px 0 0 4px;
}

button {
  padding: 0.5rem 1rem;
  border: 1px solid var(--color-accent);
  background-color: var(--color-accent);
  color: var(--color-accent-text);
  border-radius: 0 4px 4px 0;
  cursor: pointer;
  transition: background-color 0.2s;
}

button:hover {
  background-color: var(--color-accent-hover);
}

ul {
  list-style-type: none;
  padding: 0;
  margin: 0;
}

li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem;
  border-bottom: 1px solid var(--color-border);
  cursor: pointer;
  transition: background-color 0.2s;
}

li:hover {
  background-color: var(--color-bg-item-hover);
}

li.selected {
  background-color: var(--color-bg-item-selected);
  font-weight: bold;
}

.delete-btn {
  background: none;
  border: none;
  color: var(--color-danger);
  font-size: 1.2rem;
  cursor: pointer;
  padding: 0 0.5rem;
  visibility: hidden;
  opacity: 0;
  transition: opacity 0.2s, visibility 0.2s;
}

li:hover .delete-btn {
  visibility: visible;
  opacity: 1;
}
</style>