<script lang="ts">
	import { Button, CardNS, Separator } from '@www-template/ui/components';

	const { data } = $props<{ data: { operatorCount: number; activeOperatorCount: number } }>();
</script>

<main class="space-y-6 p-8">
	<section class="space-y-2">
		<h1 class="text-3xl font-bold tracking-tight">Settings</h1>
		<p class="text-slate-600">Admin Console の権限とオペレーターを管理します。</p>
	</section>
	<CardNS.Card>
		<CardNS.CardHeader>
			<CardNS.CardTitle>Operator management</CardNS.CardTitle>
			<CardNS.CardDescription>アクティブ {data.activeOperatorCount} / 合計 {data.operatorCount} 名のオペレーターが登録されています。</CardNS.CardDescription>
		</CardNS.CardHeader>
		<CardNS.CardContent class="space-y-4">
			<p class="text-sm text-slate-600">追加・ロール変更・無効化・初回セットアップトークン再発行を行います。</p>
			<Separator />
			<Button href="/settings/operators">Manage operators</Button>
		</CardNS.CardContent>
	</CardNS.Card>
</main>
