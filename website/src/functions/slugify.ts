function slugify(url: string): string {
    return url
        .toLowerCase()
        .replace(/ /g, "_")
        .replace(/[^a-z0-9-_]/g, "");
}

export { slugify };
