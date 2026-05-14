/** Eyrie TypeScript SDK — conversation DAG client. */

export interface PromptOptions {
  model?: string;
  system_prompt?: string;
  max_tokens?: number;
  stream?: boolean;
}

export interface Node {
  id: string;
  parent_id?: string;
  root_id?: string;
  sequence: number;
  node_type: string;
  content: string;
  model?: string;
  provider?: string;
  tokens_in?: number;
  tokens_out?: number;
  created_at: string;
  title?: string;
}

export interface PromptResponse {
  content: string;
  node_id: string;
}

export interface StreamEvent {
  type: "delta" | "done" | "error";
  content?: string;
  node_id?: string;
  error?: string;
}

export class EyrieClient {
  private baseURL: string;
  private headers: Record<string, string>;

  constructor(baseURL = "http://localhost:8080", apiKey?: string) {
    this.baseURL = baseURL.replace(/\/$/, "");
    this.headers = { "Content-Type": "application/json" };
    if (apiKey) {
      this.headers["Authorization"] = `Bearer ${apiKey}`;
    }
  }

  async prompt(message: string, opts?: PromptOptions): Promise<PromptResponse> {
    const body = { message, ...opts };
    const resp = await fetch(`${this.baseURL}/prompt`, {
      method: "POST",
      headers: this.headers,
      body: JSON.stringify(body),
    });
    if (!resp.ok) throw new Error(`eyrie: ${resp.status} ${await resp.text()}`);
    return resp.json();
  }

  async promptFrom(nodeId: string, message: string, opts?: PromptOptions): Promise<PromptResponse> {
    const body = { message, ...opts };
    const resp = await fetch(`${this.baseURL}/nodes/${nodeId}/prompt`, {
      method: "POST",
      headers: this.headers,
      body: JSON.stringify(body),
    });
    if (!resp.ok) throw new Error(`eyrie: ${resp.status} ${await resp.text()}`);
    return resp.json();
  }

  async *streamPrompt(message: string, opts?: PromptOptions): AsyncGenerator<StreamEvent> {
    const body = { message, stream: true, ...opts };
    const resp = await fetch(`${this.baseURL}/prompt`, {
      method: "POST",
      headers: this.headers,
      body: JSON.stringify(body),
    });
    if (!resp.ok) throw new Error(`eyrie: ${resp.status} ${await resp.text()}`);
    const reader = resp.body!.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop()!;
      for (const line of lines) {
        if (line.startsWith("data: ")) {
          yield JSON.parse(line.slice(6));
        }
      }
    }
  }

  async listConversations(): Promise<Node[]> {
    const resp = await fetch(`${this.baseURL}/nodes`, { headers: this.headers });
    if (!resp.ok) throw new Error(`eyrie: ${resp.status}`);
    return resp.json();
  }

  async getNode(nodeId: string): Promise<Node> {
    const resp = await fetch(`${this.baseURL}/nodes/${nodeId}`, { headers: this.headers });
    if (!resp.ok) throw new Error(`eyrie: ${resp.status}`);
    return resp.json();
  }

  async getTree(nodeId: string): Promise<Node[]> {
    const resp = await fetch(`${this.baseURL}/nodes/${nodeId}/tree`, { headers: this.headers });
    if (!resp.ok) throw new Error(`eyrie: ${resp.status}`);
    return resp.json();
  }

  async deleteNode(nodeId: string): Promise<void> {
    const resp = await fetch(`${this.baseURL}/nodes/${nodeId}`, { method: "DELETE", headers: this.headers });
    if (!resp.ok) throw new Error(`eyrie: ${resp.status}`);
  }

  async createAlias(nodeId: string, alias: string): Promise<void> {
    const resp = await fetch(`${this.baseURL}/nodes/${nodeId}/aliases/${alias}`, { method: "PUT", headers: this.headers });
    if (!resp.ok) throw new Error(`eyrie: ${resp.status}`);
  }

  async deleteAlias(alias: string): Promise<void> {
    const resp = await fetch(`${this.baseURL}/aliases/${alias}`, { method: "DELETE", headers: this.headers });
    if (!resp.ok) throw new Error(`eyrie: ${resp.status}`);
  }

  async health(): Promise<{ status: string }> {
    const resp = await fetch(`${this.baseURL}/health`);
    if (!resp.ok) throw new Error(`eyrie: ${resp.status}`);
    return resp.json();
  }
}
