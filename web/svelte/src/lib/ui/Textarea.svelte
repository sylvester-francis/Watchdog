<script lang="ts">
  interface Props {
    value?: string;
    rows?: number;
    placeholder?: string;
    disabled?: boolean;
    name?: string;
    id?: string;
  }

  let {
    value = $bindable(''),
    rows = 3,
    placeholder,
    disabled = false,
    name,
    id,
  }: Props = $props();
</script>

<textarea bind:value {rows} {placeholder} {disabled} {name} {id}
          class="bg-card-elevated border border-border rounded px-3 py-2 text-foreground"></textarea>
