<script lang="ts">
  interface Props {
    checked?: boolean;
    disabled?: boolean;
    name?: string;
    id?: string;
  }

  let {
    checked = $bindable(false),
    disabled = false,
    name,
    id,
  }: Props = $props();
</script>

<input type="checkbox" bind:checked {disabled} {name} {id} class="accent-accent" />
