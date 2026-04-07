export interface ConnectorTextBlock {
    type: 'connector_text';
    text: string;
}
export interface ConnectorTextDelta {
    type: 'connector_text_delta';
    text: string;
}
export declare function isConnectorTextBlock(block: unknown): block is ConnectorTextBlock;
//# sourceMappingURL=connector.d.ts.map