/**
 * Admin frontend domain layer の境界を表す型です。
 *
 * auth / account / operator orchestration は同じ domain package の各 module が担当し、
 * この型は `app -> domain -> api` の物理境界を lint / tsconfig から確認しやすくするために残します。
 */
export type AdminDomainLayerBoundary = 'admin-domain';
