import { defineCollection, z } from "astro:content";

const linksCollection = defineCollection({
	type: "data",
	schema: z.object({
		title: z.string(),
		href: z.string().url(),
		icon: z.string().optional(),
		section: z.enum([
			"Monitoring",
			"Productivity",
			"Utilities",
			"Content",
			"Smart Home",
			"External",
		]),
		order: z.number().int().positive(),
	}),
});

export const collections = {
	links: linksCollection,
};
