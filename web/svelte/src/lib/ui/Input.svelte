<script lang="ts">
  interface Props {
    value?: string;
    type?: 'text' | 'email' | 'password' | 'url' | 'number' | 'search';
    placeholder?: string;
    disabled?: boolean;
    readonly?: boolean;
    name?: string;
    id?: string;
  }

  let {
    value = $bindable(''),
    type = 'text',
    placeholder,
    disabled = false,
    readonly = false,
    name,
    id,
  }: Props = $props();
</script>

<input bind:value {type} {placeholder} {disabled} {readonly} {name} {id}
       class="bg-card-elevated border border-border rounded px-3 py-2 text-foreground" />
