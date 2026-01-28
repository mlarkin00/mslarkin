import { PropertyValues } from "lit";
import { Root } from "./root.js";
import * as Primitives from "@a2ui/web_core/types/primitives";
export declare class MultipleChoice extends Root {
    #private;
    accessor description: string | null;
    accessor options: {
        label: Primitives.StringValue;
        value: string;
    }[];
    accessor selections: Primitives.StringValue | string[];
    static styles: import("lit").CSSResult[];
    protected willUpdate(changedProperties: PropertyValues<this>): void;
    render(): import("lit-html").TemplateResult<1>;
}
//# sourceMappingURL=multiple-choice.d.ts.map