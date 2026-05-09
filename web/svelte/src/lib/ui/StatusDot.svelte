<script lang="ts">
  type Status = 'up' | 'down' | 'warn' | 'unknown';

  interface Props {
    status: Status;
    pulse?: boolean;
  }

  let { status, pulse = false }: Props = $props();
</script>

<span data-status={status} aria-label="status: {status}"
      class="ui-status-dot inline-block w-2 h-2 rounded-full"
      class:animate-pulse-dot={pulse}></span>
