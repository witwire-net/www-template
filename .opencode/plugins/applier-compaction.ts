/** セッション履歴からApplier起動元を判定するために必要なメッセージ情報です。 */
interface SessionMessage {
  info: { role: string; agent?: string };
}

/** 圧縮プラグインが利用するOpenCodeクライアントの局所契約です。 */
interface PluginClient {
  session: {
    messages(input: { path: { id: string } }): Promise<{ data?: SessionMessage[] }>;
  };
}

/** このプラグインが登録するフックだけを表す局所契約です。 */
type ApplierCompactionPlugin = (input: { client: PluginClient }) => Promise<{
  'chat.message': (input: { agent?: string; sessionID: string }) => Promise<void>;
  'experimental.session.compacting': (
    input: { sessionID: string },
    output: { context: string[] }
  ) => Promise<void>;
}>;

const APPLIER_AGENT = 'openspec/applier';
const APPLIER_COMPACTION_CONTEXT = `
This is an openspec/applier session. Preserve the latest "## Agent Delegation Timeline" block in the compaction summary with its exact Revision, Change, CLI State, execution lines, task order, agents, states, dependencies, evidence, facilitator cycle, verdict, and fix owners.

Keep completed, active, blocked, and planned delegations distinct. Preserve retained facilitator finding identifiers and their assigned fix owners. If the latest value cannot be established from the conversation, write UNKNOWN instead of inferring progress. The compacted session must be able to resume by updating this timeline before the next delegation.
`.trim();

/** Applierとして確認できたセッションを圧縮時の識別用に保持します。 */
const activeApplierSessions = new Set<string>();

/** Applierの圧縮要約へ最新の委任タイムラインを引き継ぎます。 */
const applierCompactionPlugin = (({ client }) =>
  Promise.resolve({
    'chat.message': ({ agent, sessionID }) => {
      if (agent === APPLIER_AGENT) activeApplierSessions.add(sessionID);
      return Promise.resolve();
    },
    'experimental.session.compacting': async ({ sessionID }, output) => {
      if (!activeApplierSessions.has(sessionID)) {
        try {
          const response = await client.session.messages({ path: { id: sessionID } });
          const isApplierSession = response.data?.some(
            ({ info }) => info.role === 'user' && info.agent === APPLIER_AGENT
          );
          if (isApplierSession !== true) return;
          activeApplierSessions.add(sessionID);
        } catch {
          return;
        }
      }
      output.context.push(APPLIER_COMPACTION_CONTEXT);
    },
  })) satisfies ApplierCompactionPlugin;

export default applierCompactionPlugin;
