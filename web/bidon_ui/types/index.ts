export { ResourceLink } ;

declare global {
  type SomeType = [boolean, string, number];

  interface ResourceLink {
    basePath: string;
    textField: string;
    dataField?: string;
  }
}
