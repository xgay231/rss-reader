<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue';

const sources = ref([]);
const groups = ref([]);
const newSourceUrl = ref('');
const selectedSourceId = ref(null);
const starredCount = ref(0);
const newGroupName = ref('');
const editingGroupId = ref(null);
const editingGroupName = ref('');
const collapsedGroups = ref({});
const openMenuSourceId = ref(null);
const showSubmenu = ref(null);

// Drag state
const draggingType = ref(null); // 'group' or 'source'
const draggingId = ref(null);
const dragOverType = ref(null);
const dragOverId = ref(null);

const emit = defineEmits(['source-selected']);

// Close menu when clicking outside
const handleClickOutside = (event) => {
  if (!event.target.closest('.source-item') && !event.target.closest('.dropdown-menu')) {
    openMenuSourceId.value = null;
  }
};

// Fetch all groups and sources when the component is mounted
onMounted(async () => {
  await Promise.all([fetchGroups(), fetchSources(), fetchStarredCount()]);
  document.addEventListener('click', handleClickOutside);
});

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside);
});

// Group drag handlers
const onGroupDragStart = (e, group) => {
  draggingType.value = 'group';
  draggingId.value = group.id;
  e.dataTransfer.effectAllowed = 'move';
  e.dataTransfer.setData('text/plain', group.id);
};

const onGroupDragOver = (e, group) => {
  e.preventDefault();
  if (draggingType.value === 'group' && draggingId.value !== group.id) {
    dragOverType.value = 'group';
    dragOverId.value = group.id;
  }
};

const onGroupDrop = async (e, targetGroup) => {
  e.preventDefault();
  if (draggingType.value === 'group' && draggingId.value !== targetGroup.id) {
    const fromIndex = groups.value.findIndex(g => g.id === draggingId.value);
    const toIndex = groups.value.findIndex(g => g.id === targetGroup.id);
    const [removed] = groups.value.splice(fromIndex, 1);
    groups.value.splice(toIndex, 0, removed);
  } else if (draggingType.value === 'source') {
    // Move source to target group
    const source = sources.value.find(s => s.id === draggingId.value);
    if (source && source.groupId !== targetGroup.id) {
      try {
        const response = await fetch(`/api/sources/${source.id}/group`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ groupId: targetGroup.id }),
        });
        if (response.ok) {
          source.groupId = targetGroup.id;
          sources.value = [...sources.value];
        }
      } catch (error) {
        console.error('Failed to move source to group:', error);
      }
    }
  }
  draggingType.value = null;
  draggingId.value = null;
  dragOverType.value = null;
  dragOverId.value = null;
};

const onGroupDragEnd = () => {
  draggingType.value = null;
  draggingId.value = null;
  dragOverType.value = null;
  dragOverId.value = null;
};

const onGroupDropToUngrouped = async (e) => {
  e.preventDefault();
  if (draggingType.value === 'source') {
    const source = sources.value.find(s => s.id === draggingId.value);
    if (source && source.groupId) {
      try {
        const response = await fetch(`/api/sources/${source.id}/group`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ groupId: '' }),
        });
        if (response.ok) {
          source.groupId = '';
          sources.value = [...sources.value];
        }
      } catch (error) {
        console.error('Failed to move source to ungrouped:', error);
      }
    }
  }
  draggingType.value = null;
  draggingId.value = null;
  dragOverType.value = null;
  dragOverId.value = null;
};

// Source drag handlers
const onSourceDragStart = (e, source) => {
  e.stopPropagation();
  draggingType.value = 'source';
  draggingId.value = source.id;
  e.dataTransfer.effectAllowed = 'move';
  e.dataTransfer.setData('text/plain', source.id);
};

const onSourceDragEnd = () => {
  draggingType.value = null;
  draggingId.value = null;
  dragOverType.value = null;
  dragOverId.value = null;
};

const fetchGroups = async () => {
  try {
    const response = await fetch('/api/groups');
    if (response.ok) {
      const data = await response.json();
      groups.value = data || [];
    }
  } catch (error) {
    console.error('Failed to fetch groups:', error);
  }
};

const fetchSources = async () => {
  try {
    const response = await fetch('/api/sources');
    if (!response.ok) {
      throw new Error('Network response was not ok');
    }
    const data = await response.json();
    sources.value = data || [];
  } catch (error) {
    console.error('Failed to fetch sources:', error);
  }
};

const fetchStarredCount = async () => {
  try {
    const starResponse = await fetch('/api/articles/starred');
    if (starResponse.ok) {
      const starredArticles = await starResponse.json();
      starredCount.value = starredArticles.length;
    }
  } catch (error) {
    console.error('Failed to fetch starred count:', error);
  }
};

// Grouped sources: sources grouped by their groupId
const groupedSources = computed(() => {
  const result = {};
  groups.value.forEach(group => {
    result[group.id] = sources.value.filter(s => s.groupId === group.id);
  });
  // Also get ungrouped sources (handles undefined, null, empty string, and empty ObjectID)
  const emptyObjectID = '000000000000000000000000';
  result['_ungrouped'] = sources.value.filter(s => !s.groupId || s.groupId === '' || s.groupId === emptyObjectID);
  return result;
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
    newSourceUrl.value = '';
  } catch (error) {
    console.error('Error adding source:', error);
    alert('Failed to add feed source. Check the URL and try again.');
  }
};

// Add a new group
const addGroup = async () => {
  if (!newGroupName.value.trim()) {
    return;
  }
  try {
    const response = await fetch('/api/groups', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ name: newGroupName.value }),
    });

    if (!response.ok) {
      throw new Error('Failed to add group');
    }

    const newGroup = await response.json();
    groups.value.push(newGroup);
    newGroupName.value = '';
  } catch (error) {
    console.error('Error adding group:', error);
    alert('Failed to add group.');
  }
};

// Delete a group
const deleteGroup = async (groupId, event) => {
  event.stopPropagation();
  if (!confirm('Delete this group? Sources will become ungrouped.')) {
    return;
  }
  try {
    const response = await fetch(`/api/groups/${groupId}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      throw new Error('Failed to delete group');
    }
    groups.value = groups.value.filter(g => g.id !== groupId);
    delete collapsedGroups.value[groupId];
    // Refresh sources to reflect groupId changes
    await fetchSources();
  } catch (error) {
    console.error('Error deleting group:', error);
    alert('Failed to delete group.');
  }
};

// Start editing a group
const startEditGroup = (group, event) => {
  event.stopPropagation();
  editingGroupId.value = group.id;
  editingGroupName.value = group.name;
};

// Save edited group
const saveEditGroup = async () => {
  if (!editingGroupName.value.trim()) {
    cancelEditGroup();
    return;
  }
  try {
    const response = await fetch(`/api/groups/${editingGroupId.value}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ name: editingGroupName.value }),
    });

    if (!response.ok) {
      throw new Error('Failed to update group');
    }

    const group = groups.value.find(g => g.id === editingGroupId.value);
    if (group) {
      group.name = editingGroupName.value;
    }
    cancelEditGroup();
  } catch (error) {
    console.error('Error updating group:', error);
    alert('Failed to update group.');
  }
};

const cancelEditGroup = () => {
  editingGroupId.value = null;
  editingGroupName.value = '';
};

const selectSource = (source) => {
  selectedSourceId.value = source.id;
  openMenuSourceId.value = null; // Close dropdown menu
  emit('source-selected', source);
};

const selectStarred = async () => {
  selectedSourceId.value = 'starred';
  openMenuSourceId.value = null; // Close dropdown menu
  emit('source-selected', { id: 'starred', name: '收藏夹' });
  await fetchStarredCount();
};

// Refresh starred count
const refreshStarredCount = async () => {
  await fetchStarredCount();
};

defineExpose({ refreshStarredCount });

// Delete a source
const deleteSource = async (sourceId, event) => {
  event.stopPropagation();
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
    if (selectedSourceId.value === sourceId) {
      selectedSourceId.value = null;
      emit('source-selected', null);
    }
  } catch (error) {
    console.error('Error deleting source:', error);
    alert('Failed to delete feed source.');
  }
};

// Assign source to a group
const assignSourceToGroup = async (source, groupId, event) => {
  event.stopPropagation();
  openMenuSourceId.value = null; // Close dropdown menu
  try {
    const response = await fetch(`/api/sources/${source.id}/group`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ groupId: groupId }),
    });

    if (!response.ok) {
      throw new Error('Failed to assign source to group');
    }

    // Update local state
    source.groupId = groupId || null;
    // Force reactivity
    sources.value = [...sources.value];
  } catch (error) {
    console.error('Error assigning source to group:', error);
    alert('Failed to assign source to group.');
  }
};

// Toggle menu for a source
const toggleMenu = (sourceId) => {
  openMenuSourceId.value = openMenuSourceId.value === sourceId ? null : sourceId;
  showSubmenu.value = null;
};

// Toggle group collapse
const toggleGroupCollapse = (groupId) => {
  collapsedGroups.value[groupId] = !collapsedGroups.value[groupId];
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
    <div class="add-group-form">
      <input
        type="text"
        v-model="newGroupName"
        placeholder="New group name"
        @keyup.enter="addGroup"
      />
      <button @click="addGroup">Add Group</button>
    </div>

    <ul class="source-list">
      <!-- Starred section -->
      <li
        class="starred-item"
        :class="{ selected: selectedSourceId === 'starred' }"
        @click="selectStarred"
      >
        <span>★ 收藏夹</span>
        <span class="star-count" v-if="starredCount > 0">{{ starredCount }}</span>
      </li>

      <!-- Groups -->
      <li
        v-for="group in groups"
        :key="group.id"
        class="group-item"
        :class="{ 'drag-over': dragOverType === 'group' && dragOverId === group.id }"
        draggable="true"
        @dragstart="onGroupDragStart($event, group)"
        @dragover="onGroupDragOver($event, group)"
        @drop="onGroupDrop($event, group)"
        @dragend="onGroupDragEnd"
      >
        <div
          class="group-header"
          @click="toggleGroupCollapse(group.id)"
        >
          <span class="collapse-icon">{{ collapsedGroups[group.id] ? '▶' : '▼' }}</span>
          <span class="group-name" v-if="editingGroupId !== group.id">{{ group.name }}</span>
          <input
            v-else
            v-model="editingGroupName"
            class="group-edit-input"
            @keyup.enter="saveEditGroup"
            @keyup.escape="cancelEditGroup"
            @blur="saveEditGroup"
            @click.stop
          />
          <div class="group-actions" @click.stop>
            <button class="action-btn" @click="startEditGroup(group, $event)">✎</button>
            <button class="action-btn delete-btn" @click="deleteGroup(group.id, $event)">×</button>
          </div>
        </div>
        <ul class="source-list nested" v-if="!collapsedGroups[group.id]">
          <li
            v-for="source in groupedSources[group.id]"
            :key="source.id"
            class="source-item"
            :class="{ selected: source.id === selectedSourceId }"
            draggable="true"
            @dragstart="onSourceDragStart($event, source)"
            @dragend="onSourceDragEnd"
            @click="selectSource(source)"
          >
            <span class="source-name">{{ source.name }}</span>
            <div class="more-btn" @click.stop="toggleMenu(source.id)">⋮</div>
            <!-- Dropdown menu -->
            <div class="dropdown-menu" v-if="openMenuSourceId === source.id">
              <div class="menu-item has-submenu" @mouseenter="showSubmenu = source.id" @mouseleave="showSubmenu = null">
                <span>分组</span>
                <span class="arrow">›</span>
                <!-- Submenu -->
                <div class="submenu" v-if="showSubmenu === source.id">
                  <div class="menu-item" @click="assignSourceToGroup(source, '', $event)">无分组</div>
                  <div
                    v-for="g in groups"
                    :key="g.id"
                    class="menu-item"
                    :class="{ active: source.groupId === g.id }"
                    @click="assignSourceToGroup(source, g.id, $event)"
                  >
                    {{ g.name }}
                  </div>
                </div>
              </div>
              <div class="menu-item danger" @click="deleteSource(source.id, $event)">删除</div>
            </div>
          </li>
        </ul>
      </li>

      <!-- Ungrouped sources -->
      <li
        class="group-item ungrouped"
        v-if="groupedSources['_ungrouped'] && groupedSources['_ungrouped'].length > 0"
        :class="{ 'drag-over': dragOverType === 'group' && dragOverId === '_ungrouped' }"
        draggable="true"
        @dragstart="onGroupDragStart($event, { id: '_ungrouped', name: '未分组' })"
        @dragover="onGroupDragOver($event, { id: '_ungrouped' })"
        @drop="onGroupDropToUngrouped($event)"
        @dragend="onGroupDragEnd"
      >
        <div
          class="group-header"
          @click="toggleGroupCollapse('_ungrouped')"
        >
          <span class="collapse-icon">{{ collapsedGroups['_ungrouped'] ? '▶' : '▼' }}</span>
          <span class="group-name">未分组</span>
        </div>
        <ul class="source-list nested" v-if="!collapsedGroups['_ungrouped']">
          <li
            v-for="source in groupedSources['_ungrouped']"
            :key="source.id"
            class="source-item"
            :class="{ selected: source.id === selectedSourceId }"
            draggable="true"
            @dragstart="onSourceDragStart($event, source)"
            @dragend="onSourceDragEnd"
            @click="selectSource(source)"
          >
            <span class="source-name">{{ source.name }}</span>
            <div class="more-btn" @click.stop="toggleMenu(source.id)">⋮</div>
            <!-- Dropdown menu -->
            <div class="dropdown-menu" v-if="openMenuSourceId === source.id">
              <div class="menu-item has-submenu" @mouseenter="showSubmenu = source.id" @mouseleave="showSubmenu = null">
                <span>分组</span>
                <span class="arrow">›</span>
                <!-- Submenu -->
                <div class="submenu" v-if="showSubmenu === source.id">
                  <div class="menu-item" @click="assignSourceToGroup(source, '', $event)">无分组</div>
                  <div
                    v-for="g in groups"
                    :key="g.id"
                    class="menu-item"
                    :class="{ active: source.groupId === g.id }"
                    @click="assignSourceToGroup(source, g.id, $event)"
                  >
                    {{ g.name }}
                  </div>
                </div>
              </div>
              <div class="menu-item danger" @click="deleteSource(source.id, $event)">删除</div>
            </div>
          </li>
        </ul>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.source-list-container {
  padding: 1rem;
  border-right: 1px solid var(--color-border);
  height: 100vh;
  overflow: visible;
  background-color: var(--color-bg-pane);
}

h2 {
  margin-top: 0;
  font-size: 1.2rem;
  color: var(--color-text-primary);
}

.add-source-form, .add-group-form {
  display: flex;
  margin-bottom: 0.5rem;
  position: relative;
  z-index: 20;
}

.add-group-form {
  margin-bottom: 1rem;
}

input[type="text"] {
  flex: 1;
  min-width: 0;
  padding: 0.5rem;
  border: 1px solid var(--color-border);
  border-radius: 4px 0 0 4px;
  position: relative;
  z-index: 20;
}

button {
  padding: 0.5rem 1rem;
  border: 1px solid var(--color-accent);
  background-color: var(--color-accent);
  color: var(--color-accent-text);
  border-radius: 0 4px 4px 0;
  cursor: pointer;
  transition: background-color 0.2s;
  position: relative;
  z-index: 20;
}

button:hover {
  background-color: var(--color-accent-hover);
}

.source-list {
  list-style-type: none;
  padding: 0;
  margin: 0;
}

.source-list.nested {
  padding-left: 1rem;
}

li {
  position: relative;
}

.starred-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem;
  border-bottom: 1px solid var(--color-border);
  cursor: pointer;
  transition: background-color 0.2s;
  color: var(--color-accent);
}

.starred-item:hover, .starred-item.selected {
  background-color: var(--color-bg-item-hover);
}

.group-item {
  border-bottom: 1px solid var(--color-border);
}

.group-item.drag-over {
  border: 2px dashed var(--color-accent);
  background-color: var(--color-bg-item-hover);
}

.group-item.ungrouped .group-header {
  color: var(--color-text-secondary);
}

.group-item.ungrouped .collapse-icon {
  opacity: 0.5;
}

.group-header {
  display: flex;
  align-items: center;
  padding: 0.75rem;
  cursor: pointer;
  transition: background-color 0.2s;
  gap: 0.25rem;
}

.group-header:hover {
  background-color: var(--color-bg-item-hover);
}

.collapse-icon {
  font-size: 0.7rem;
  width: 1rem;
  color: var(--color-text-secondary);
}

.group-name {
  flex: 1;
  font-weight: 500;
}

.group-edit-input {
  flex: 1;
  padding: 0.2rem;
  font-size: inherit;
  border: 1px solid var(--color-accent);
  border-radius: 2px;
}

.group-actions {
  display: flex;
  gap: 0.25rem;
  opacity: 0;
  transition: opacity 0.2s;
}

.group-header:hover .group-actions {
  opacity: 1;
}

.source-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem;
  border-bottom: 1px solid var(--color-border);
  cursor: pointer;
  transition: background-color 0.2s;
}

.source-item.drag-over {
  border: 2px dashed var(--color-accent);
  background-color: var(--color-bg-item-hover);
}

.source-item:hover, .source-item.selected {
  background-color: var(--color-bg-item-hover);
}

.source-item.selected {
  background-color: var(--color-bg-item-selected);
  font-weight: bold;
}

.source-item.dragging {
  opacity: 0.5;
  background-color: var(--color-bg-item-hover);
}

.source-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.more-btn {
  padding: 0.25rem 0.5rem;
  cursor: pointer;
  color: var(--color-text-secondary);
  font-size: 1.2rem;
  border-radius: 4px;
}

.more-btn:hover {
  background-color: var(--color-bg-item-hover);
  color: var(--color-text-primary);
}

.dropdown-menu {
  position: absolute;
  right: 0;
  top: 100%;
  background: var(--color-bg-pane);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
  z-index: 100;
  min-width: 120px;
}

.menu-item {
  padding: 0.6rem 1rem;
  cursor: pointer;
  white-space: nowrap;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.menu-item:hover {
  background-color: var(--color-bg-item-hover);
}

.menu-item.danger {
  color: var(--color-danger);
}

.menu-item.danger:hover {
  background-color: #fee;
}

.menu-item.active {
  color: var(--color-accent);
  font-weight: bold;
}

.has-submenu {
  position: relative;
}

.arrow {
  font-size: 1.2rem;
}

.submenu {
  position: absolute;
  left: 100%;
  top: 0;
  background: var(--color-bg-pane);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.15);
  min-width: 120px;
  z-index: 200;
}

.star-count {
  background-color: var(--color-accent);
  color: var(--color-accent-text);
  font-size: 0.75rem;
  padding: 0.1rem 0.4rem;
  border-radius: 10px;
  min-width: 1.2rem;
  text-align: center;
}
</style>